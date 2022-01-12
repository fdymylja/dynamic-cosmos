package signing

import "fmt"

type Keystore interface {
	Sign(addr string, bytes []byte) (signed []byte, err error)
}

type errorKeyStore struct {
}

func (errorKeyStore) Sign(_ string, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("unable to sign")
}
