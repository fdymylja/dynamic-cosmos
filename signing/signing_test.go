package signing

import (
	"context"
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

func TestAccountInfo(t *testing.T) {
	const endpoint = "34.94.191.28:9090"

	reflectionRemote, err := codec.NewGRPCReflectionRemote(endpoint)
	require.NoError(t, err)

	cdc := codec.NewCodec(reflectionRemote)
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.ForceCodec(cdc.GRPCCodec())))
	require.NoError(t, err)
	s := NewDirectSigner(conn, cdc, errorKeyStore{})
	pubKey, sequence, err := s.authInfo(context.Background(), "osmo1yc26vffx8mmdvj97eauwzr6lcu0l2jdjfl069e")
	require.NoError(t, err)

	t.Logf("%s %v", pubKey, sequence)
}
