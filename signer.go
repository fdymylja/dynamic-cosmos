package dynamic

import "google.golang.org/protobuf/types/known/anypb"

type Signer interface {
	Sign(addr string, bytes []byte) (signature []byte, err error)
	PubKeyForAddr(addr string) (*anypb.Any, error)
}
