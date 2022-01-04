package dynamic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	const endpoint = "34.94.191.28:9090"
	grpcRem, err := NewGRPCReflectionRemote(endpoint)
	require.NoError(t, err)

	c, err := NewClient(context.Background(), grpcRem, "34.94.191.28:9090", "")
	require.NoError(t, err)

	for k, svc := range c.ModuleQueries {
		t.Logf("%s", k)
		t.Logf(svc.ParentFile().Path())
		t.Log(svc.ParentFile().FullName())
	}

	for name, msg := range c.Messages {
		t.Logf("message typeURL: %s, name: %s", name, msg.Descriptor().Name())
	}

	// try with cache remote
	fds, err := c.Registry.Save()
	require.NoError(t, err)

	cacheRemo := NewCacheRemote(fds)
	multi := NewMultiRemote(cacheRemo, grpcRem)

	c, err = NewClient(context.Background(), multi, "34.94.191.28:9090", "")
	require.NoError(t, err)
}
