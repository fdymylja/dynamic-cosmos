package dynamic

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/api/cosmos/base/reflection/v2alpha1"
	"github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"strings"
)

type Client struct {
	App           *reflectionv2alpha1.AppDescriptor
	Registry      *Registry
	Codec         *Codec
	ModuleQueries map[protoreflect.FullName]protoreflect.ServiceDescriptor
	Messages      map[protoreflect.FullName]protoreflect.MessageType

	tm   *http.HTTP
	grpc grpc.ClientConnInterface
}

func NewClient(ctx context.Context, remote RemoteRegistry, grpcEndpoint string, tmEndpoint string) (*Client, error) {
	conn, err := grpc.DialContext(ctx, grpcEndpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	cosmosReflection := reflectionv2alpha1.NewReflectionServiceClient(conn)

	authn, err := cosmosReflection.GetAuthnDescriptor(ctx, &reflectionv2alpha1.GetAuthnDescriptorRequest{})
	if err != nil {
		return nil, err
	}
	chain, err := cosmosReflection.GetChainDescriptor(ctx, &reflectionv2alpha1.GetChainDescriptorRequest{})
	if err != nil {
		return nil, err
	}
	codec, err := cosmosReflection.GetCodecDescriptor(ctx, &reflectionv2alpha1.GetCodecDescriptorRequest{})
	if err != nil {
		return nil, err
	}
	conf, err := cosmosReflection.GetConfigurationDescriptor(ctx, &reflectionv2alpha1.GetConfigurationDescriptorRequest{})
	if err != nil {
		return nil, err
	}
	query, err := cosmosReflection.GetQueryServicesDescriptor(ctx, &reflectionv2alpha1.GetQueryServicesDescriptorRequest{})
	if err != nil {
		return nil, err
	}
	tx, err := cosmosReflection.GetTxDescriptor(ctx, &reflectionv2alpha1.GetTxDescriptorRequest{})
	if err != nil {
		return nil, err
	}

	app := &reflectionv2alpha1.AppDescriptor{
		Authn:         authn.Authn,
		Chain:         chain.Chain,
		Codec:         codec.Codec,
		Configuration: conf.Config,
		QueryServices: query.Queries,
		Tx:            tx.Tx,
	}

	reg := NewRegistry(remote)
	c := &Client{
		App:           app,
		Registry:      reg,
		Codec:         NewCodec(reg),
		ModuleQueries: map[protoreflect.FullName]protoreflect.ServiceDescriptor{},
		Messages:      map[protoreflect.FullName]protoreflect.MessageType{},
		tm:            nil,
	}

	err = c.prepare()
	if err != nil {
		return nil, err
	}

	tm, err := http.New(tmEndpoint, "/websocket")
	if err != nil {
		return nil, err
	}

	c.tm = tm

	// now we recreate a new grpc with a custom codec that uses our proto marshaler and unmarshaler
	// which enable us to resolve and handle message dynamically without having knowledge of those
	err = conn.Close()
	if err != nil {
		return nil, err
	}
	conn, err = grpc.DialContext(
		ctx, grpcEndpoint,
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodec(c.Codec.GRPCCodec()),
		),
	)
	if err != nil {
		return nil, err
	}

	c.grpc = conn
	return c, nil
}

func (c *Client) prepare() error {
	// fetch query services
	for _, svc := range c.App.QueryServices.QueryServices {
		desc, err := c.Registry.FindDescriptorByName(protoreflect.FullName(svc.Fullname))
		if err != nil {
			return fmt.Errorf("unable to fetch information for query service %s: %w", svc.Fullname, err)
		}

		c.ModuleQueries[protoreflect.FullName(svc.Fullname)] = desc.(protoreflect.ServiceDescriptor)
	}
	// fetch messages
	for _, msg := range c.App.Tx.Msgs {
		message := protoreflect.FullName(msg.MsgTypeUrl)
		if i := strings.LastIndexByte(msg.MsgTypeUrl, '/'); i >= 0 {
			message = message[i+len("/"):]
		}

		md, err := c.Registry.FindDescriptorByName(message)
		if err != nil {
			return fmt.Errorf("unable to fetch information for message %s: %w", msg.MsgTypeUrl, err)
		}

		c.Messages[md.FullName()] = dynamicpb.NewMessageType(md.(protoreflect.MessageDescriptor))
	}

	return nil
}

func (c *Client) Query(ctx context.Context, method string, req, resp proto.Message) (err error) {
	return c.grpc.Invoke(ctx, method, req, resp)
}
