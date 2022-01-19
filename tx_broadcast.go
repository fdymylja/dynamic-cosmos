package dynamic

import (
	"context"
	"fmt"
	abciv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/abci/v1beta1"
	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/cosmos/cosmos-sdk/api/tendermint/abci"
	"github.com/fdymylja/dynamic-cosmos/tx"
	"github.com/tendermint/tendermint/crypto"
)

// BroadcastTx is an alias of tx.Response and
// identifies a transaction which was broadcast
type BroadcastTx = tx.Response

func NewBroadcastTx(ctx context.Context, bytes []byte, mode txv1beta1.BroadcastMode, txSvc txv1beta1.ServiceClient, watcher *tx.Watcher) (<-chan *BroadcastTx, error) {
	switch mode {
	case txv1beta1.BroadcastMode_BROADCAST_MODE_BLOCK:
		// this will return only if success
		resp, err := txSvc.BroadcastTx(ctx, &txv1beta1.BroadcastTxRequest{
			TxBytes: bytes,
			Mode:    mode,
		})
		if err != nil {
			return nil, err
		}
		if resp.TxResponse.Code != 0 {
			return nil, newBroadcastError(resp.TxResponse)
		}

		c := make(chan *BroadcastTx, 1)
		c <- &BroadcastTx{
			Bytes: bytes,
			Result: &abci.ResponseDeliverTx{
				Code:      resp.TxResponse.Code,
				Data:      []byte(resp.TxResponse.Data),
				Log:       resp.TxResponse.RawLog,
				Info:      resp.TxResponse.Info,
				GasWanted: resp.TxResponse.GasWanted,
				GasUsed:   resp.TxResponse.GasUsed,
				Events:    resp.TxResponse.Events,
				Codespace: resp.TxResponse.Codespace,
			},
			Block: resp.TxResponse.Height,
			Index: 0, // TODO(this is unfilled)
		}
		return c, nil
	case txv1beta1.BroadcastMode_BROADCAST_MODE_SYNC:
		hash := fmt.Sprintf("%X", crypto.Sha256(bytes))
		c, err := watcher.Watch(ctx, hash)
		// this will return only checktx response
		resp, err := txSvc.BroadcastTx(ctx, &txv1beta1.BroadcastTxRequest{
			TxBytes: bytes,
			Mode:    mode,
		})
		if err != nil {
			return nil, err
		}
		// check if code is ok
		if resp.TxResponse.Code != 0 {
			return nil, newBroadcastError(resp.TxResponse)
		}

		return c, nil

	default:
		return nil, fmt.Errorf("unsupported broadcast mode: %s", mode)
	}
}

func newBroadcastError(resp *abciv1beta1.TxResponse) *BroadcastTxError {
	return &BroadcastTxError{Response: resp}
}

// BroadcastTxError identifies an error whilst broadcasting a TX.
type BroadcastTxError struct {
	Response *abciv1beta1.TxResponse
}

func (e *BroadcastTxError) Error() string {
	return fmt.Sprintf("tx with hash %s failed: %s", e.Response.Txhash, e.Response.RawLog)
}
