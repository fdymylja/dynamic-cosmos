package dynamic

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/keys"
	"github.com/coinbase/rosetta-sdk-go/types"
	bankv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/bank/v1beta1"
	basev1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/v1beta1"
	secp256k12 "github.com/cosmos/cosmos-sdk/api/cosmos/crypto/secp256k1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/removeme/bech32"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"golang.org/x/crypto/ripemd160"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func getCacheRemote(t *testing.T) codec.RemoteRegistry {
	f, err := os.Open("./data/osmosis.proto.json")
	require.NoError(t, err)
	defer f.Close()
	fdSetBytes, err := ioutil.ReadAll(f)
	require.NoError(t, err)

	fdSet := new(descriptorpb.FileDescriptorSet)
	require.NoError(t, protojson.Unmarshal(fdSetBytes, fdSet))

	return codec.NewCacheRemote(fdSet)
}

var _ Signer = (*mapSigner)(nil)

type mapSigner struct {
	pairs map[string]*keys.KeyPair
}

func (m mapSigner) PubKeyForAddr(addr string) (*anypb.Any, error) {
	pair, exists := m.pairs[addr]
	if !exists {
		return nil, fmt.Errorf("unknown addr: %s", addr)
	}
	key := &secp256k12.PubKey{
		Key: pair.PublicKey.Bytes,
	}

	keyBytes, err := proto.Marshal(key)
	if err != nil {
		return nil, err
	}

	return &anypb.Any{
		TypeUrl: "/" + (string)(key.ProtoReflect().Descriptor().FullName()),
		Value:   keyBytes,
	}, nil
}

func (m mapSigner) Sign(addr string, bytes []byte) (signature []byte, err error) {
	key, exist := m.pairs[addr]
	if !exist {
		return nil, fmt.Errorf("unknown signer: %s", addr)
	}

	signer, err := key.Signer()
	if err != nil {
		return nil, err
	}

	sig, err := signer.Sign(&types.SigningPayload{
		Bytes:         crypto.Sha256(bytes),
		SignatureType: types.Ecdsa,
	}, types.Ecdsa)

	if err != nil {
		return nil, err
	}

	return sig.Bytes, nil
}

func TestTx_Sign(t *testing.T) {
	const privKeyHex = "fill your private key hex here"

	addr := derive(t, "osmo", privKeyHex)

	privKey, err := keys.ImportPrivateKey(privKeyHex, types.Secp256k1)
	client, err := NewClient(context.Background(), getCacheRemote(t), "34.94.191.28:9090", "tcp://34.94.191.28:26657",
		WithAuthenticationOption(
			WithSigner(&mapSigner{map[string]*keys.KeyPair{addr: privKey}})))
	require.NoError(t, err)

	tx := client.NewTx()

	require.NoError(t, tx.AddMsg(&bankv1beta1.MsgSend{
		FromAddress: addr,
		ToAddress:   "osmo1v8ujerydzj6z0ga7zqf53eh9849l6pq8uu72vr",
		Amount: []*basev1beta1.Coin{
			{
				Denom:  "uosmo",
				Amount: "1",
			},
		},
	}))

	tx.AddSignerByAddr(addr)
	tx.SetFeePayer(addr)
	tx.SetFee(&basev1beta1.Coin{Denom: "uosmo", Amount: "1"})
	tx.SetGasLimit(500000)
	signed, err := tx.Sign(context.Background())
	require.NoError(t, err)

	t.Logf("%s", signed)

	b, err := client.Codec.MarshalProto(signed)

	t.Logf("%x", b)

	res, err := client.tm.BroadcastTxCommit(context.Background(), b)
	require.NoError(t, err)
	t.Logf("%#v", res)
}

func derive(t *testing.T, bech32Prefix, privKeyHex string) string {

	pair, err := keys.ImportPrivateKey(privKeyHex, types.Secp256k1)
	require.NoError(t, err)

	require.Len(t, pair.PublicKey.Bytes, 33)

	sha := sha256.Sum256(pair.PublicKey.Bytes)
	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha[:]) // does not error
	addrBytes := hasherRIPEMD160.Sum(nil)

	bechifiedAddr, err := bech32.ConvertAndEncode(bech32Prefix, addrBytes)
	require.NoError(t, err)

	return bechifiedAddr
}
