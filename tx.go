package dynamic

import (
	"context"
	"fmt"

	basev1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/v1beta1"
	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/protoutil"
	"github.com/fdymylja/dynamic-cosmos/signing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func NewTx(cdc *codec.Codec, supportedMsgs map[protoreflect.FullName]struct{}, chainID string, signeInfoProvider SignerInfoProvider, signer Signer) *Tx {
	return &Tx{
		supported: supportedMsgs,
		chainID:   chainID,
		tx: &txv1beta1.Tx{
			Body: &txv1beta1.TxBody{},
			AuthInfo: &txv1beta1.AuthInfo{
				SignerInfos: nil,
				Fee:         &txv1beta1.Fee{},
				// Tip:         &txv1beta1.Tip{},
			},
			Signatures: nil,
		},
		cdc:              cdc,
		signersAddr:      map[string]struct{}{},
		authInfoProvider: signeInfoProvider,
		signer:           signer,
	}
}

type Tx struct {
	supported map[protoreflect.FullName]struct{}
	chainID   string

	tx  *txv1beta1.Tx
	cdc *codec.Codec

	signersAddr      map[string]struct{}
	authInfoProvider SignerInfoProvider
	signer           Signer
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

		any.TypeUrl = "/" + string(protoutil.FullNameFromURL(any.TypeUrl)) // TODO(fdymylja): fixme

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
	if t.tx.AuthInfo.Fee.Payer == addr {
		return
	}

	t.signersAddr[addr] = struct{}{}
}

// SetFeePayer sets the Tx fee payer. It also
// adds fee payer as a signer of the transaction.
func (t *Tx) SetFeePayer(addr string) {
	t.tx.AuthInfo.Fee.Payer = addr
}

func (t *Tx) SetFee(coins ...*basev1beta1.Coin) {
	t.tx.AuthInfo.Fee.Amount = coins
}

func (t *Tx) SetGasLimit(limit uint64) {
	t.tx.AuthInfo.Fee.GasLimit = limit
}

func (t *Tx) Sign(ctx context.Context) (*txv1beta1.TxRaw, error) {
	if err := t.valid(); err != nil {
		return nil, fmt.Errorf("invalid tx: %w", err)
	}

	// we check if user has set both fee payer as signer too which is not required
	if _, exists := t.signersAddr[t.tx.AuthInfo.Fee.Payer]; exists {
		delete(t.signersAddr, t.tx.AuthInfo.Fee.Payer)
	}

	// populate account info
	signers := make([]string, 0, len(t.signersAddr)+1) // signers plus fee payer
	signers = append(signers, t.tx.AuthInfo.Fee.Payer)
	for signer := range t.signersAddr {
		signers = append(signers, signer)
	}

	signerInfos := make([]*SignerInfoExtended, 0, len(signers))
	for _, signer := range signers {
		info, err := t.authInfoProvider.SignerInfo(ctx, signer)
		if err != nil {
			return nil, fmt.Errorf("unable to get auth info for address %s: %w", signer, err)
		}

		// NOTE: if pubkey is not set we need to fetch it somewhere
		// this happens for accounts interacting for the first time
		// with a chain
		if info.SignerInfo.PublicKey == nil {
			pubKey, err := t.signer.PubKeyForAddr(signer)
			if err != nil {
				return nil, fmt.Errorf("unable to get pubkey for address %s: %w", signer, err)
			}
			info.SignerInfo.PublicKey = pubKey
		}

		signerInfos = append(signerInfos, info)
	}

	// set signer info TODO(fdymylja): we know array length
	for _, info := range signerInfos {
		t.tx.AuthInfo.SignerInfos = append(t.tx.AuthInfo.SignerInfos, info.SignerInfo)
	}

	signatures := make([][]byte, len(signerInfos))

	for i, info := range signerInfos {

		signature, err := signing.Direct(t.cdc, t.tx.Body, t.tx.AuthInfo, t.chainID, info.AccountNumber)
		if err != nil {
			return nil, fmt.Errorf("unable to compute signature: %w", err)
		}

		signedDoc, err := t.signer.Sign(signers[i], signature)
		if err != nil {
			return nil, err
		}
		signatures[i] = signedDoc
	}

	t.tx.Signatures = signatures

	return txToTxRaw(t.cdc, t.tx)
}

func (t *Tx) Broadcast(ctx context.Context, mode txv1beta1.BroadcastMode) (*BroadcastTx, error) {
	panic("impl")
}

func (t *Tx) valid() error {
	if t.tx.AuthInfo.Fee.Payer == "" {
		return fmt.Errorf("no fee payer specified")
	}

	if t.tx.AuthInfo.Fee.GasLimit == 0 {
		return fmt.Errorf("no gas limit specified")
	}

	if len(t.tx.Body.Messages) == 0 {
		return fmt.Errorf("no messages in transaction")
	}

	if len(t.tx.AuthInfo.Fee.Amount) == 0 {
		return fmt.Errorf("no fee amounts specified")
	}

	for i, c := range t.tx.AuthInfo.Fee.Amount {
		if c.Amount == "" {
			return fmt.Errorf("no amount specified for fee coin at index %d", i)
		}

		if c.Denom == "" {
			return fmt.Errorf("no denom specified for fee coin at index %d", i)
		}
	}

	return nil
}

type BroadcastTx struct {
}

func txToTxRaw(cdc *codec.Codec, tx *txv1beta1.Tx) (*txv1beta1.TxRaw, error) {
	bodyBytes, err := cdc.MarshalProto(tx.Body)
	if err != nil {
		return nil, err
	}

	authBytes, err := cdc.MarshalProto(tx.AuthInfo)
	if err != nil {
		return nil, err
	}

	return &txv1beta1.TxRaw{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authBytes,
		Signatures:    tx.Signatures,
	}, nil
}
