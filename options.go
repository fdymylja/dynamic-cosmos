package dynamic

import (
	"context"
	"fmt"

	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/tx"

	reflectionv2alpha1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/reflection/v2alpha1"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/protoutil"
	"github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func newOptions(grpcEndpoint, tmEndpoint string) *options {
	return &options{
		grpcEndpoint:       grpcEndpoint,
		tendermintEndpoint: tmEndpoint,
		auth: &authenticationOptions{
			signer:             nil,
			signerInfoProvider: nil,
			supportedMessages:  map[protoreflect.FullName]struct{}{},
		},
	}
}

// options defines the options of a client
type options struct {
	grpcEndpoint       string
	tendermintEndpoint string

	appDesc *reflectionv2alpha1.AppDescriptor
	remote  codec.ProtoFileRegistry
	auth    *authenticationOptions
}

// setup sets up the *Client
func (o *options) setup(ctx context.Context) (*Client, error) {
	if o.tendermintEndpoint == "" {
		return nil, fmt.Errorf("no tendermint endpoint set")
	}
	if o.grpcEndpoint == "" {
		return nil, fmt.Errorf("no grpc endpoint set")
	}

	// we check if remote is set, if it's not set we default
	// to the grpc registry remote
	if o.remote == nil {
		remote, err := codec.NewGRPCReflectionProtoFileRegistry(o.grpcEndpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to set up grpc remote protofile registry: %w", err)
		}
		o.remote = remote
	}

	// setup codec
	cdc := codec.NewCodec(o.remote)

	// dial grpc connection
	conn, err := grpc.DialContext(ctx, o.grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(cdc.GRPCCodec())))
	if err != nil {
		return nil, err
	}

	// we need to fetch the app descriptor if it was not set
	if o.appDesc == nil {
		err = o.setAppDesc(ctx, conn)
		if err != nil {
			return nil, fmt.Errorf("unable to setup app descriptor: %w", err)
		}
	}

	// setup addresses
	addresses := o.addresses(o.appDesc.Configuration)

	// set up authentication options
	err = o.auth.setup(cdc, conn, o.appDesc.Tx)
	if err != nil {
		return nil, fmt.Errorf("unable to setup authentication options: %w", err)
	}

	// set up tendermint
	tm, err := http.New(o.tendermintEndpoint, "/websocket")
	if err != nil {
		return nil, err
	}
	err = tm.Start()
	if err != nil {
		return nil, err
	}

	txWatcher, err := tx.DialWatcher(ctx, tm)
	if err != nil {
		return nil, err
	}

	return &Client{
		App:         o.appDesc,
		Codec:       cdc,
		Addresses:   addresses,
		dynQueriers: nil,
		dynMessage:  nil,
		tm:          tm,
		grpc:        conn,
		watcher:     txWatcher,
		txSvc:       txv1beta1.NewServiceClient(conn),
		authOpt:     o.auth,
	}, nil
}

func (o *options) setAppDesc(ctx context.Context, conn grpc.ClientConnInterface) error {
	rc := reflectionv2alpha1.NewReflectionServiceClient(conn)
	authn, err := rc.GetAuthnDescriptor(ctx, &reflectionv2alpha1.GetAuthnDescriptorRequest{})
	if err != nil {
		return err
	}
	chain, err := rc.GetChainDescriptor(ctx, &reflectionv2alpha1.GetChainDescriptorRequest{})
	if err != nil {
		return err
	}
	codecInfo, err := rc.GetCodecDescriptor(ctx, &reflectionv2alpha1.GetCodecDescriptorRequest{})
	if err != nil {
		return err
	}
	conf, err := rc.GetConfigurationDescriptor(ctx, &reflectionv2alpha1.GetConfigurationDescriptorRequest{})
	if err != nil {
		return err
	}
	query, err := rc.GetQueryServicesDescriptor(ctx, &reflectionv2alpha1.GetQueryServicesDescriptorRequest{})
	if err != nil {
		return err
	}
	tx, err := rc.GetTxDescriptor(ctx, &reflectionv2alpha1.GetTxDescriptorRequest{})
	if err != nil {
		return err
	}

	o.appDesc = &reflectionv2alpha1.AppDescriptor{
		Authn:         authn.Authn,
		Chain:         chain.Chain,
		Codec:         codecInfo.Codec,
		Configuration: conf.Config,
		QueryServices: query.Queries,
		Tx:            tx.Tx,
	}

	return nil
}

func (o *options) addresses(configuration *reflectionv2alpha1.ConfigurationDescriptor) *Addresses {
	return NewAddresses(configuration.Bech32AccountAddressPrefix)
}

// DialOption defines a Client Dial option
type DialOption func(options *options)

// WithRemoteRegistry allows to setup a custom protobuf file registry
func WithRemoteRegistry(registry codec.ProtoFileRegistry) DialOption {
	return func(options *options) {
		options.remote = registry
	}
}

// WithAppDescriptor allows to provide the application descriptor in case
// it was cached beforehand to speed up the dial process.
func WithAppDescriptor(desc *reflectionv2alpha1.AppDescriptor) DialOption {
	return func(options *options) {
		options.appDesc = desc
	}
}

type authenticationOptions struct {
	signer             Signer
	signerInfoProvider SignerInfoProvider
	supportedMessages  map[protoreflect.FullName]struct{}
}

func (o *authenticationOptions) setup(cdc *codec.Codec, conn grpc.ClientConnInterface, desc *reflectionv2alpha1.TxDescriptor) error {
	// setup supported chain messages
	for _, msg := range desc.Msgs {
		o.supportedMessages[protoutil.FullNameFromURL(msg.MsgTypeUrl)] = struct{}{}
	}
	// no signer provided, which means this might be only a query setup
	// so we setup an erroring signer
	if o.signer == nil {
		o.signer = erroringSigner{}
	}

	if o.signerInfoProvider == nil {
		o.signerInfoProvider = newAuthModuleSignerInfoProvider(cdc, conn)
	}

	return nil
}

type AuthenticationOption func(opt *authenticationOptions)

// WithAuthenticationOptions sets the authentication settings for the client.
func WithAuthenticationOptions(authOpts ...AuthenticationOption) DialOption {
	return func(options *options) {
		for _, authOpt := range authOpts {
			authOpt(options.auth)
		}
	}
}

// WithSigner sets the signature provider for tx requests
func WithSigner(s Signer) AuthenticationOption {
	return func(opt *authenticationOptions) {
		opt.signer = s
	}
}

// WithSignerInfoProvider sets the SignerInfoProvider for Client
func WithSignerInfoProvider(s SignerInfoProvider) AuthenticationOption {
	return func(opt *authenticationOptions) {
		opt.signerInfoProvider = s
	}
}

var _ Signer = (*erroringSigner)(nil)

type erroringSigner struct{}

func (e erroringSigner) Sign(_ string, _ []byte) (signature []byte, err error) {
	return nil, fmt.Errorf("this setup does not support sending transactions")
}

func (e erroringSigner) PubKeyForAddr(_ string) (*anypb.Any, error) {
	return nil, fmt.Errorf("this setup does not support sending transactions")
}

var _ grpc.ClientConnInterface = (*erroringConn)(nil)

// erroringConn is a grpc.ClientConnInterface that returns the provided error
type erroringConn struct {
	err error
}

func (e erroringConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	return e.err
}

func (e erroringConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, e.err
}
