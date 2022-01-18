package tx

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/rpc/client/http"
	tmtypes "github.com/tendermint/tendermint/types"
)

func TestNewTxWatcher(t *testing.T) {
	h, err := http.New("tcp://34.94.191.28:26657", "/websocket")
	require.NoError(t, h.Start())
	require.NoError(t, err)
	x, err := NewWatcher(context.Background(), h)
	require.NoError(t, err)
	require.Nil(t, x)

	time.Sleep(155 * time.Second)
}

func TestNewX(t *testing.T) {
	h, err := http.New("tcp://34.94.191.28:26657", "/websocket")
	require.NoError(t, err)
	require.NoError(t, h.Start())
	const newTxQuery = "tm.event='" + tmtypes.EventNewBlock + "'"

	x, err := h.Subscribe(context.Background(), "asdas", newTxQuery)
	require.NoError(t, err)

	for {
		a := <-x
		log.Printf("%#v", a)
	}
}
