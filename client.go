package dynamic

import (
	"context"
	"fmt"
	"io"

	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/tx"

	"github.com/cosmos/cosmos-sdk/api/cosmos/base/reflection/v2alpha1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/protoutil"
	"github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Client struct {
	App       *reflectionv2alpha1.AppDescriptor
	Codec     *codec.Codec
	Addresses *Addresses

	dynQueriers map[protoreflect.FullName]protoreflect.ServiceDescriptor
	dynMessage  map[protoreflect.FullName]protoreflect.MessageType

	tm      *http.HTTP
	grpc    grpc.ClientConnInterface
	watcher *tx.Watcher
	txSvc   txv1beta1.ServiceClient

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
	return NewTx(c.Codec, c.authOpt.supportedMessages, c.App.Chain.Id, c.authOpt.signerInfoProvider, c.authOpt.signer, c.watcher, c.txSvc)
}

func (c *Client) ClientConn() grpc.ClientConnInterface {
	return c.grpc
}

func (c *Client) Close() error {
	var reasons []error

	err := c.tm.Stop()
	if err != nil {
		reasons = append(reasons, err)
	}

	c.watcher.Stop()
	if closer, ok := c.grpc.(io.Closer); ok {
		err = closer.Close()
		if err != nil {
			reasons = append(reasons, err)
		}
	}

	err = c.Codec.Registry.Remote().Close() // TODO better
	if err != nil {
		reasons = append(reasons, err)
	}

	if len(reasons) == 0 {
		return nil
	}

	return reasons[0] // TODO better
}
