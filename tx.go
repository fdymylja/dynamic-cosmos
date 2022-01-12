package dynamic

import (
	"context"
	"fmt"
	basev1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/v1beta1"
	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func NewTx(cdc *codec.Codec, supportedMsgs map[protoreflect.FullName]struct{}) *Tx {
	return &Tx{
		tx: &txv1beta1.Tx{
			Body: &txv1beta1.TxBody{},
			AuthInfo: &txv1beta1.AuthInfo{
				SignerInfos: nil,
				Fee:         &txv1beta1.Fee{},
				Tip:         &txv1beta1.Tip{},
			},
			Signatures: nil,
		},
		supported:   supportedMsgs,
		cdc:         cdc,
		signersAddr: map[string]struct{}{},
		feePayer:    "",
	}
}

type Tx struct {
	tx        *txv1beta1.Tx
	supported map[protoreflect.FullName]struct{}
	cdc       *codec.Codec

	signersAddr map[string]struct{}
	feePayer    string
}

func (t *Tx) AddMsgs(msgs ...proto.Message) error {
	// check if the chain supports all the provided messages
	for _, m := range msgs {
		if _, supported := t.supported[m.ProtoReflect().Descriptor().FullName()]; !supported {
			return fmt.Errorf("msg %s is not supported by the chain", m.ProtoReflect().Descriptor().FullName())
		}

		any := new(anypb.Any)
		err := anypb.MarshalFrom(any, m, t.cdc.ProtoOptions().Marshal)
		if err != nil {
			return fmt.Errorf("unable to marshal %s as anypb.Any: %w", m.ProtoReflect().Descriptor().FullName(), err)
		}

		t.tx.Body.Messages = append(t.tx.Body.Messages, any)
	}

	return nil
}

func (t *Tx) AddMsg(m proto.Message) error {
	return t.AddMsgs(m)
}

func (t *Tx) SetMemo(memo string) {
	t.tx.Body.Memo = memo
}

func (t *Tx) SetTimeoutHeight(height uint64) {
	t.tx.Body.TimeoutHeight = height
}

func (t *Tx) AddSignerByPubKey(pubKey []byte) {
	// forces us to depend on the sdk
	// but the idea is simple, we know the
	// chain config, and hence bech32 prefixes
	// so we can just compute the pubkey to address here.
	panic("impl")
}

func (t *Tx) AddSignerByAddr(addr string) {
	// TODO(fdymylja): should we error in case same addr is set twice?
	if t.feePayer == addr {
		return
	}

	t.signersAddr[addr] = struct{}{}
}

func (t *Tx) SetFeePayer(addr string) {
	t.tx.AuthInfo.Fee.Payer = addr
}

func (t *Tx) SetFee(coins ...*basev1beta1.Coin) {
	t.tx.AuthInfo.Fee.Amount = coins
}

func (t *Tx) SetGasLimit(limit uint64) {
	t.tx.AuthInfo.Fee.GasLimit = limit
}

func (t *Tx) Broadcast(ctx context.Context, mode txv1beta1.BroadcastMode) (*BroadcastTx, error) {
	panic("impl")
}

type BroadcastTx struct {
}
