package main

import (
	"io/ioutil"
	"log"
	"net/http"

	txv1beta1 "github.com/cosmos/cosmos-sdk/api/cosmos/tx/v1beta1"
	"github.com/fdymylja/dynamic-cosmos/codec"
)

// this converts json txs to proto, the example handles one chain only
// but it's possible to create multiple codecs (one for each chain)
// even add chains on the fly to handle json to proto conversion, etc.

func main() {
	const grpcEndpoint = ""

	remote, err := codec.NewGRPCReflectionRemote(grpcEndpoint)
	if err != nil {
		panic(err)
	}

	cdc := codec.NewCodec(remote)
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		jsonBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			return
		}

		tx := new(txv1beta1.Tx)
		err = cdc.UnmarshalProtoJSON(jsonBody, tx)
		if err != nil {
			return
		}

		protoBody, err := cdc.MarshalProto(tx)
		if err != nil {
			return
		}

		_, err = writer.Write(protoBody)
		if err != nil {
			return
		}

		return
	})))
}
