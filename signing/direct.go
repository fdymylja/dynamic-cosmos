package signing

import (
	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
)

// Direct provides the required signature bytes using sign mode direct specification.
func Direct(cdc *codec.Codec, txBody *txv1beta1.TxBody, authInfo *txv1beta1.AuthInfo, chainID string, accountNumber uint64) ([]byte, error) {
	txBodyBytes, err := cdc.MarshalProto(txBody)
	if err != nil {
		return nil, err
	}

	authInfoBytes, err := cdc.MarshalProto(authInfo)
	if err != nil {
		return nil, err
	}

	doc := &txv1beta1.SignDoc{
		BodyBytes:     txBodyBytes,
		AuthInfoBytes: authInfoBytes,
		ChainId:       chainID,
		AccountNumber: accountNumber,
	}

	return cdc.MarshalProto(doc)
}
