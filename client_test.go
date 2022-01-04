package dynamic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient(context.Background(), "34.94.191.28:9090", "")
	require.NoError(t, err)

	for k, svc := range c.ModuleQueries {
		t.Logf("%s", k)
		t.Logf(svc.ParentFile().Path())
		t.Log(svc.ParentFile().FullName())
	}
}
