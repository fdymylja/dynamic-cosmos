package main

import (
	"github.com/tendermint/tendermint/libs/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const EnvAddr = "addr"
const DefaultAddr = ":5555"

func main() {
	mcf := newMultiChainFaucet()

	addr := DefaultAddr
	if envAddr, ok := os.LookupEnv(EnvAddr); ok {
		addr = envAddr
	}

	mux := new(http.ServeMux)
	mux.HandleFunc("/newChain", func(writer http.ResponseWriter, request *http.Request) {
		b, err := ioutil.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("can't read body: " + err.Error()))
			return
		}

		x := new(NewChainRequest)
		err = json.Unmarshal(b, x)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("can't json unmarshal body: " + err.Error()))
			return
		}

		err = mcf.Add(request.Context(), x)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("can't add chain: " + err.Error()))
			return
		}

		_, _ = writer.Write([]byte("chain added"))
	})

	mux.HandleFunc("/send", func(writer http.ResponseWriter, request *http.Request) {
		b, err := ioutil.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("can't read body: " + err.Error()))
			return
		}

		x := new(SendFundsRequest)
		err = json.Unmarshal(b, x)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("can't json unmarshal body: " + err.Error()))
			return
		}

		err = mcf.Send(request.Context(), x)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte("can't send funds: " + err.Error()))
			return
		}

		_, _ = writer.Write([]byte("funds sent"))
	})

	log.Printf("serving at: %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
