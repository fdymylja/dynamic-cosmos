package tx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	tmrpc "github.com/tendermint/tendermint/rpc/client"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Watcher struct {
	txClient txv1beta1.ServiceClient

	doneOnce *sync.Once
	done     chan struct{}

	cdc *codec.Codec

	subs   map[string][]chan *txv1beta1.Tx
	addSub chan struct {
		c    chan *txv1beta1.Tx
		hash string
	}
}

// Watch returns a channel that sends a txv1beta1.Tx, once its found.
// Contract: the tx is readonly.
func (w *Watcher) Watch(ctx context.Context, hash string) (<-chan *txv1beta1.Tx, error) {
	c := make(chan *txv1beta1.Tx, 1)
	select {
	case w.addSub <- struct {
		c    chan *txv1beta1.Tx
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
			// TODO(fdymylja): proper cleanup
			return
		case c := <-w.addSub:
			w.subs[c.hash] = append(w.subs[c.hash], c.c)
		case newTx := <-txs:
			log.Printf("DSADS")
			b, err := json.Marshal(newTx)
			if err != nil {
				panic(err)
			}
			panic(b)
			log.Printf("%s", b)
			txHash, exists := newTx.Events[tmtypes.TxHashKey]
			if !exists {
				panic(fmt.Errorf("invalid tx event format: %#v", newTx))
			}

			watchers, exists := w.subs[txHash[0]]
			if !exists {
				break
			}
			_ = watchers
		}
	}
}

func NewWatcher(ctx context.Context, sub tmrpc.EventsClient) (*Watcher, error) {
	const newTxQuery = "tm.event='" + tmtypes.EventTx + "'"
	ws, err := sub.Subscribe(ctx, "dynamisdasdsadasc-cosmos", newTxQuery)
	if err != nil {
		return nil, err
	}

	txWatcher := &Watcher{
		txClient: nil,
		doneOnce: new(sync.Once),
		done:     make(chan struct{}),
		cdc:      nil,
		subs:     map[string][]chan *txv1beta1.Tx{},
		addSub: make(chan struct {
			c    chan *txv1beta1.Tx
			hash string
		}),
	}

	go txWatcher.loop(ws)

	return nil, nil
}
