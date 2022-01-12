package signing

import (
	"context"
	"fmt"
	authv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/auth/v1beta1"
	basev1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/v1beta1"
	signingv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/protoutil"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func NewDirectSigner(conn grpc.ClientConnInterface, codec *codec.Codec, provider Keystore) *DirectSigner {
	return &DirectSigner{
		codec:    codec,
		auth:     authv1beta1.NewQueryClient(conn),
		provider: provider,
	}
}

type DirectSigner struct {
	codec    *codec.Codec
	auth     authv1beta1.QueryClient
	provider Keystore
	chainID  string
}

func (s *DirectSigner) Sign(ctx context.Context, addr string, msgs ...proto.Message) (signedTx []byte, err error) {
	return s.MultiSign(ctx, []string{addr}, msgs...)
}

func (s *DirectSigner) MultiSign(ctx context.Context, addrs []string, msgs ...proto.Message) (signedTx []byte, err error) {

	signerInfos := make([]*txv1beta1.SignerInfo, len(addrs))

	for i, addr := range addrs {
		pubKey, sequence, err := s.authInfo(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("unable to get auth info for %s: %w", addr, err)
		}

		signerInfo := &txv1beta1.SignerInfo{
			PublicKey: pubKey,
			ModeInfo: &txv1beta1.ModeInfo{
				Sum: &txv1beta1.ModeInfo_Single_{
					Single: &txv1beta1.ModeInfo_Single{
						Mode: signingv1beta1.SignMode_SIGN_MODE_DIRECT,
					},
				},
			},
			Sequence: sequence,
		}

		signerInfos[i] = signerInfo
	}

	_ = &txv1beta1.AuthInfo{
		SignerInfos: signerInfos,
		Fee: &txv1beta1.Fee{
			Amount:   []*basev1beta1.Coin{},
			GasLimit: 100000000,
			Payer:    addrs[0],
		},
	}

	_ = &txv1beta1.TxBody{
		Messages:                    nil,
		Memo:                        "",
		TimeoutHeight:               0,
		ExtensionOptions:            nil,
		NonCriticalExtensionOptions: nil,
	}

	_ = txv1beta1.TxRaw{
		BodyBytes:     nil,
		AuthInfoBytes: nil,
		Signatures:    nil,
	}

	_ = txv1beta1.SignDoc{
		BodyBytes:     nil,
		AuthInfoBytes: nil,
		ChainId:       "",
		AccountNumber: 0,
	}

	panic("impl")
}

func (s *DirectSigner) authInfo(ctx context.Context, addr string) (*anypb.Any, uint64, error) {
	accountResponse, err := s.auth.Account(ctx, &authv1beta1.QueryAccountRequest{Address: addr})
	if err != nil {
		return nil, 0, err
	}

	accInfoProvider, exists := accountInfoProvider[protoutil.FullNameFromURL(accountResponse.Account.TypeUrl)]
	if !exists {
		return nil, 0, fmt.Errorf("cannot extract pubkey and sequence from: %s", accountResponse.Account.TypeUrl)
	}

	return accInfoProvider(s.codec, accountResponse.Account)
}
