package dynamic

import (
	"context"
	"testing"

	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Test_authSignerInfoProvider_SignerInfo(t *testing.T) {
	ctx := context.Background()
	grpcRemote, err := codec.NewGRPCReflectionProtoFileRegistry("34.94.191.28:9090")
	require.NoError(t, err)
	cdc := codec.NewCodec(grpcRemote)
	conn, err := grpc.DialContext(ctx, "34.94.191.28:9090", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.ForceCodec(cdc.GRPCCodec())))
	require.NoError(t, err)
	a := newAuthModuleSignerInfoProvider(cdc, conn)

	signerInfo, err := a.SignerInfo(ctx, "osmo1g95nzqyvd27mhwek7wfdqkr8l2v329s4dk590l")
	require.NoError(t, err)

	t.Logf("%s", signerInfo)
}
