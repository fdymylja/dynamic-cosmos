package dynamic

import (
	"context"
	"os"
	"testing"

	queryv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/base/query/v1beta1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/gov/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"

	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	const endpoint = "34.94.191.28:9090"

	c, err := Dial(context.Background(), "34.94.191.28:9090", "")
	require.NoError(t, err)
	for k, svc := range c.dynQueriers {
		t.Logf("%s", k)
		t.Logf(svc.ParentFile().Path())
		t.Log(svc.ParentFile().FullName())
	}

	for name, msg := range c.dynMessage {
		t.Logf("message typeURL: %s, name: %s", name, msg.Descriptor().Name())
	}

	// try with cache remote
	fds, err := c.Codec.Registry.Save()
	require.NoError(t, err)

	cacheRemo := codec.NewCacheProtoFileRegistry(fds)
	multi := codec.NewMultiProtoFileRegistry(cacheRemo, c.Codec.Registry.Remote())

	c, err = Dial(context.Background(), "34.94.191.28:9090", "", WithRemoteRegistry(multi))
	require.NoError(t, err)

	jsonBytes, err := c.Codec.MarshalProtoJSON(fds)
	require.NoError(t, err)

	f, err := os.OpenFile("./data/osmosis.proto.json", os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(t, err)
	defer f.Close()

	_, err = f.Write(jsonBytes)
	require.NoError(t, err)
}

func TestInvokeProposal(t *testing.T) {
	const endpoint = "34.94.191.28:9090"
	const tmEndpoint = "tcp://34.94.191.28:26657"

	c, err := Dial(context.Background(), endpoint, tmEndpoint)
	require.NoError(t, err)

	respt, err := c.Codec.Registry.FindMessageByName("cosmos.gov.v1beta1.QueryProposalsResponse")
	require.NoError(t, err)

	resp := respt.New()
	err = c.DynamicQuery(context.Background(), "/cosmos.gov.v1beta1.Query/Proposals", &govv1beta1.QueryProposalsRequest{
		ProposalStatus: govv1beta1.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED,
		Voter:          "",
		Depositor:      "",
		Pagination: &queryv1beta1.PageRequest{
			Key:        nil,
			Offset:     0,
			Limit:      10000000000,
			CountTotal: false,
			Reverse:    false,
		},
	}, resp.Interface())
	require.NoError(t, err)

	jsonBytes, err := c.Codec.MarshalProtoJSON(resp.Interface())
	require.NoError(t, err)

	t.Logf("%s", jsonBytes)
}
