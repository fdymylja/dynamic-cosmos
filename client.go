package dynamic

import (
	"context"
	"fmt"

	authv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/auth/v1beta1"
	bankv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/bank/v1beta1"
	"github.com/cosmos/cosmos-sdk/api/cosmos/base/reflection/v2alpha1"
	distributionv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/distribution/v1beta1"
	evidencev1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/evidence/v1beta1"
	feegrantv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/feegrant/v1beta1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/gov/v1beta1"
	mintv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/mint/v1beta1"
	paramsv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/params/v1beta1"
	slashingv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/slashing/v1beta1"
	stakingv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/staking/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/protoutil"
	"github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Queriers struct {
	Auth         authv1beta1.QueryClient
	Bank         bankv1beta1.QueryClient
	Distribution distributionv1beta1.QueryClient
	Evidence     evidencev1beta1.QueryClient
	FeeGrant     feegrantv1beta1.QueryClient
	Gov          govv1beta1.QueryClient
	Mint         mintv1beta1.QueryClient
	Params       paramsv1beta1.QueryClient
	Slashing     slashingv1beta1.QueryClient
	Staking      stakingv1beta1.QueryClient
}

type Client struct {
	App      *reflectionv2alpha1.AppDescriptor
	Codec    *codec.Codec
	Queriers Queriers

	dynQueriers map[protoreflect.FullName]protoreflect.ServiceDescriptor
	dynMessage  map[protoreflect.FullName]protoreflect.MessageType

	tm   *http.HTTP
	grpc grpc.ClientConnInterface

	authOpt *authenticationOptions
}

func Dial(ctx context.Context, grpcEndpoint string, tmEndpoint string, dialOptions ...DialOption) (*Client, error) {
	opts := newOptions(grpcEndpoint, tmEndpoint)
	for _, o := range dialOptions {
		o(opts)
	}

	return opts.setup(ctx)
}

// TODO(fdymylja): decide what to do with this
func (c *Client) prepare() error {
	// fetch query services
	for _, svc := range c.App.QueryServices.QueryServices {
		desc, err := c.Codec.Registry.FindDescriptorByName(protoreflect.FullName(svc.Fullname))
		if err != nil {
			return fmt.Errorf("unable to fetch information for query service %s: %w", svc.Fullname, err)
		}

		c.dynQueriers[protoreflect.FullName(svc.Fullname)] = desc.(protoreflect.ServiceDescriptor)
	}
	// fetch messages
	for _, msg := range c.App.Tx.Msgs {
		message := protoutil.FullNameFromURL(msg.MsgTypeUrl)

		md, err := c.Codec.Registry.FindDescriptorByName(message)
		if err != nil {
			return fmt.Errorf("unable to fetch information for message %s: %w", msg.MsgTypeUrl, err)
		}

		c.dynMessage[md.FullName()] = dynamicpb.NewMessageType(md.(protoreflect.MessageDescriptor))
	}

	return nil
}

func (c *Client) DynamicQuery(ctx context.Context, method string, req, resp proto.Message) (err error) {
	return c.grpc.Invoke(ctx, method, req, resp)
}

func (c *Client) NewTx() *Tx {
	return NewTx(c.Codec, c.authOpt.supportedMessages, c.App.Chain.Id, c.authOpt.signerInfoProvider, c.authOpt.signer)
}

func (c *Client) ClientConn() grpc.ClientConnInterface {
	return c.grpc
}
