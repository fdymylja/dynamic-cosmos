package dynamic

import (
	"context"
	"fmt"

	authv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/auth/v1beta1"
	signingv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/signing/v1beta1"
	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/protoutil"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
)

// SignerInfoProvider provides information regarding signers.
type SignerInfoProvider interface {
	SignerInfo(ctx context.Context, addr string) (*SignerInfoExtended, error)
}

// SignerInfoExtended is like txv1beta1.SignerInfo
// but contains also the account number of the account.
type SignerInfoExtended struct {
	SignerInfo    *txv1beta1.SignerInfo
	AccountNumber uint64
}

func (s SignerInfoExtended) String() string {
	return fmt.Sprintf("account_number: %d %s", s.AccountNumber, s.SignerInfo)
}

var _ SignerInfoProvider = (*authModuleSignerInfoProvider)(nil)

type authModuleSignerInfoProvider struct {
	cdc  *codec.Codec
	auth authv1beta1.QueryClient
}

func newAuthModuleSignerInfoProvider(cdc *codec.Codec, conn grpc.ClientConnInterface) *authModuleSignerInfoProvider {
	return &authModuleSignerInfoProvider{
		cdc:  cdc,
		auth: authv1beta1.NewQueryClient(conn),
	}
}

func (a authModuleSignerInfoProvider) SignerInfo(ctx context.Context, addr string) (*SignerInfoExtended, error) {
	accResp, err := a.auth.Account(ctx, &authv1beta1.QueryAccountRequest{Address: addr})
	if err != nil {
		return nil, err
	}

	switch protoutil.FullNameFromURL(accResp.Account.TypeUrl) {
	case "cosmos.auth.v1beta1.BaseAccount":
		return a.signerInfoBaseAccount(accResp.Account)
	default:
		return nil, fmt.Errorf("cannot provide signer info for account type: %s", accResp.Account.TypeUrl)
	}
}

func (a authModuleSignerInfoProvider) signerInfoBaseAccount(anyAccount *anypb.Any) (*SignerInfoExtended, error) {
	account := new(authv1beta1.BaseAccount)
	err := a.cdc.UnmarshalProto(anyAccount.Value, account)
	if err != nil {
		return nil, err
	}

	return &SignerInfoExtended{
		SignerInfo: &txv1beta1.SignerInfo{
			PublicKey: account.PubKey,
			ModeInfo: &txv1beta1.ModeInfo{
				Sum: &txv1beta1.ModeInfo_Single_{
					Single: &txv1beta1.ModeInfo_Single{
						Mode: signingv1beta1.SignMode_SIGN_MODE_DIRECT,
					},
				},
			}, // TODO(fdymylja): SignerInfoProvider should not set this.
			Sequence: account.Sequence,
		},
		AccountNumber: account.AccountNumber,
	}, nil
}
