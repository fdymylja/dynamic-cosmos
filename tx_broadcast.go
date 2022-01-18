package dynamic

import (
	"context"

	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/cosmos/cosmos-sdk/api/tendermint/abci"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/tendermint/tendermint/rpc/client/http"
)

// BroadcastTx identifies a transaction sent to the tendermint endpoint.
type BroadcastTx struct {
	tm    *http.HTTP
	raw   *txv1beta1.TxRaw
	bytes []byte

	client txv1beta1.ServiceClient
	cdc    *codec.Codec

	result chan *abci.ResponseDeliverTx
}

// Result returns the delivertx response of a transaction
func (t *BroadcastTx) Result(ctx context.Context) (*abci.ResponseDeliverTx, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-t.result:
		return res, nil
	}
}
