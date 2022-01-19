package tx

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/api/tendermint/abci"
	"github.com/hashicorp/go-uuid"
	"github.com/tendermint/tendermint/abci/types"
	"log"
	"sync"
	"time"

	tmrpc "github.com/tendermint/tendermint/rpc/client"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

const newTxQuery = "tm.event='" + tmtypes.EventTx + "'"

type Response struct {
	Bytes  []byte                  // readonly
	Result *abci.ResponseDeliverTx // readonly
	Block  int64
	Index  uint32
}

type Watcher struct {
	id string

	doneOnce *sync.Once
	done     chan struct{}

	subs   map[string][]chan *Response
	addSub chan struct {
		c    chan *Response
		hash string
	}

	client tmrpc.EventsClient // used to stop the subscription
}

// Watch returns a channel that sends a Response, once its found.
// Contract: *Response is readonly.
func (w *Watcher) Watch(ctx context.Context, hash string) (<-chan *Response, error) {
	c := make(chan *Response, 1)
	select {
	case w.addSub <- struct {
		c    chan *Response
		hash string
	}{c: c, hash: hash}:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-w.done:
		return nil, fmt.Errorf("tx: watcher is closed")
	}
}

func (w *Watcher) loop(txs <-chan coretypes.ResultEvent) {
	for {
		select {
		case <-w.done:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := w.client.Unsubscribe(ctx, w.id, newTxQuery)
			if err != nil {
				log.Printf("unable to close tendermint ws correctly: %s", err)
			}
			// TODO(fdymylja): close all channels?
			return
		case c := <-w.addSub:
			w.subs[c.hash] = append(w.subs[c.hash], c.c)
		case newTx := <-txs:
			txHash, exists := newTx.Events[tmtypes.TxHashKey]
			if !exists {
				panic(fmt.Errorf("invalid tx event format: %#v", newTx))
			}

			watchers, exists := w.subs[txHash[0]]
			if !exists {
				break
			}

			txData := newTx.Data.(tmtypes.EventDataTx)
			resp := &Response{
				Bytes:  txData.Tx,
				Result: resultProtov1toProtov2(txData.Result),
				Block:  txData.Height,
				Index:  txData.Index,
			}

			for _, watcher := range watchers {
				watcher <- resp
				close(watcher)
			}

			delete(w.subs, txHash[0])
		}
	}
}

func resultProtov1toProtov2(result types.ResponseDeliverTx) *abci.ResponseDeliverTx {
	// deep copy events
	events := make([]*abci.Event, len(result.Events))
	for i, e := range result.Events {
		// deep copy attributes
		attributes := make([]*abci.EventAttribute, len(e.Attributes))
		for j, a := range e.Attributes {
			attributes[j] = &abci.EventAttribute{
				Key:   a.Key,
				Value: a.Value,
				Index: a.Index,
			}
		}
		events[i] = &abci.Event{
			Type_:      e.Type,
			Attributes: attributes,
		}
	}
	// deep copy
	return &abci.ResponseDeliverTx{
		Code:      result.Code,
		Data:      result.Data,
		Log:       result.Log,
		Info:      result.Info,
		GasWanted: result.GasWanted,
		GasUsed:   result.GasUsed,
		Events:    events,
		Codespace: result.Codespace,
	}
}

func (w *Watcher) Stop() {
	w.doneOnce.Do(func() {
		close(w.done)
	})
}
func DialWatcher(ctx context.Context, sub tmrpc.EventsClient) (*Watcher, error) {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}

	ws, err := sub.Subscribe(ctx, id, newTxQuery)
	if err != nil {
		return nil, err
	}

	txWatcher := &Watcher{
		id:       id,
		doneOnce: new(sync.Once),
		done:     make(chan struct{}),
		subs:     map[string][]chan *Response{},
		addSub: make(chan struct {
			c    chan *Response
			hash string
		}),
		client: sub,
	}

	go txWatcher.loop(ws)

	return txWatcher, nil
}
