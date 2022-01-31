package main

import (
	"context"
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/keys"
	"github.com/coinbase/rosetta-sdk-go/types"
	bankv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/bank/v1beta1"
	basev1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/v1beta1"
	"github.com/cosmos/cosmos-sdk/api/cosmos/crypto/secp256k1"
	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/tendermint/tendermint/crypto"
	"google.golang.org/protobuf/types/known/anypb"
	"log"
	"sync"
)

type clientWithAddr struct {
	*dynamic.Client
	addr string
}

// NewChainRequest adds a new chain
type NewChainRequest struct {
	// AppIdentifier identifies the chain in a unique way
	// in the context of the faucet, used to send funds.
	AppIdentifier string `json:"app_identifier,omitempty"`
	// TendermintEndpoint is the tendermint rpc address
	TendermintEndpoint string `json:"tendermint_endpoint,omitempty"`
	// GRPCEndpoint is the gRPC address
	GRPCEndpoint string `json:"grpc_endpoint,omitempty"`
	// PrivateKey is the private key of the account which contains the funds
	PrivateKey string `json:"private_key,omitempty"`
}

// SendFundsRequest sends funds
type SendFundsRequest struct {
	Fee           *basev1beta1.Coin   `json:"fee,omitempty"`
	AppIdentifier string              `json:"app_identifier,omitempty"`
	To            string              `json:"to,omitempty"`
	Amount        []*basev1beta1.Coin `json:"amount,omitempty"`
}

func newMultiChainFaucet() *multiChainFaucet {
	return &multiChainFaucet{
		mu:      new(sync.RWMutex),
		clients: map[string]*clientWithAddr{},
	}
}

type multiChainFaucet struct {
	mu *sync.RWMutex

	clients map[string]*clientWithAddr
}

func (m *multiChainFaucet) Send(ctx context.Context, req *SendFundsRequest) error {
	client, err := m.getClient(req.AppIdentifier)
	if err != nil {
		return err
	}

	tx := client.NewTx()
	tx.SetGasLimit(100000)
	tx.SetFee(req.Fee)
	tx.SetFeePayer(client.addr)
	tx.AddSignerByAddr(client.addr)

	err = tx.AddMsg(&bankv1beta1.MsgSend{
		FromAddress: client.addr,
		ToAddress:   req.To,
		Amount:      req.Amount,
	})
	if err != nil {
		return nil
	}

	respTx, err := tx.Broadcast(ctx, txv1beta1.BroadcastMode_BROADCAST_MODE_BLOCK)
	if err != nil {
		return nil
	}

	select {
	case <-respTx:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}

}

func (m *multiChainFaucet) Add(ctx context.Context, req *NewChainRequest) error {
	// check if exists
	if m.knownChain(req.AppIdentifier) {
		return fmt.Errorf("app identifier already in use: %s", req.AppIdentifier)
	}

	signer, err := newSigner(req.PrivateKey)
	if err != nil {
		return err
	}

	log.Printf("instantiating new client for chain %#v", req)

	c, err := dynamic.Dial(ctx,
		req.GRPCEndpoint,
		req.TendermintEndpoint,
		dynamic.WithAuthenticationOptions(
			dynamic.WithSigner(signer),
		),
	)
	if err != nil {
		return err
	}

	err = signer.setAddrAndPk(c.Addresses, c.Codec)
	if err != nil {
		return err
	}

	log.Printf("new client instantiation successful, saving")
	m.mapChainToClient(req.AppIdentifier, &clientWithAddr{
		Client: c,
		addr:   signer.addr,
	})

	return nil
}

func (m *multiChainFaucet) knownChain(identifier string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.clients[identifier]
	return ok
}

func (m *multiChainFaucet) mapChainToClient(identifier string, c *clientWithAddr) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[identifier] = c
}

func (m *multiChainFaucet) getClient(identifier string) (*clientWithAddr, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.clients[identifier]
	if !ok {
		return nil, fmt.Errorf("unknown app identifier: %s", identifier)
	}

	return c, nil
}

func newSigner(pk string) (*signer, error) {
	kp, err := keys.ImportPrivateKey(pk, types.Secp256k1)
	if err != nil {
		return nil, err
	}

	return &signer{k: kp}, nil
}

type signer struct {
	addr  string
	k     *keys.KeyPair
	anyPk *anypb.Any
}

func (s signer) Sign(addr string, bytes []byte) (signature []byte, err error) {
	if s.addr != addr {
		return nil, fmt.Errorf("unknown address")
	}

	sig, err := s.k.Signer()
	if err != nil {
		return nil, err
	}

	signed, err := sig.Sign(&types.SigningPayload{
		Bytes:         crypto.Sha256(bytes),
		SignatureType: types.Ecdsa,
	}, types.Ecdsa)

	if err != nil {
		return nil, err
	}

	return signed.Bytes, nil
}

func (s signer) PubKeyForAddr(addr string) (pk *anypb.Any, err error) {
	if s.addr != addr {
		return nil, fmt.Errorf("unknown address")
	}
	if err != nil {
		return nil, err
	}

	return s.anyPk, nil
}

func (s *signer) setAddrAndPk(a *dynamic.Addresses, cdc *codec.Codec) error {
	addr, err := a.DeriveFromPubKey(s.k.PublicKey.Bytes)
	if err != nil {
		return err
	}

	s.addr = addr

	pk, err := cdc.NewAny(&secp256k1.PubKey{
		Key: s.k.PublicKey.Bytes,
	})
	if err != nil {
		return err
	}

	s.anyPk = pk
	return nil
}
