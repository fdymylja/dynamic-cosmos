package dynamic

import (
	authv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/auth/v1beta1"
	bankv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/bank/v1beta1"
	distributionv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/distribution/v1beta1"
	evidencev1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/evidence/v1beta1"
	feegrantv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/feegrant/v1beta1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/gov/v1beta1"
	mintv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/mint/v1beta1"
	paramsv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/params/v1beta1"
	slashingv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/slashing/v1beta1"
	stakingv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/staking/v1beta1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var moduleQueriersFD = []protoreflect.FileDescriptor{
	authv1beta1.File_cosmos_auth_v1beta1_query_proto,
	bankv1beta1.File_cosmos_bank_v1beta1_query_proto,
	distributionv1beta1.File_cosmos_distribution_v1beta1_query_proto,
	evidencev1beta1.File_cosmos_evidence_v1beta1_query_proto,
	feegrantv1beta1.File_cosmos_feegrant_v1beta1_query_proto,
	govv1beta1.File_cosmos_gov_v1beta1_query_proto,
	mintv1beta1.File_cosmos_mint_v1beta1_query_proto,
	paramsv1beta1.File_cosmos_params_v1beta1_query_proto,
	slashingv1beta1.File_cosmos_slashing_v1beta1_query_proto,
	stakingv1beta1.File_cosmos_staking_v1beta1_query_proto,
}
