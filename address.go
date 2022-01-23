package dynamic

import (
	"fmt"

	"github.com/fdymylja/dynamic-cosmos/internal/removeme/bech32"
)

// AddressHuman is a string type alias for a human-readable address representation.
type AddressHuman = string

// AddressRaw is a []byte type alias for a raw bytes address representation.
type AddressRaw = []byte

// NewAddresses instantiates a new *Addresses instance.
func NewAddresses(bech32Prefix string) *Addresses {
	return &Addresses{bech32Prefix: bech32Prefix}
}

// Addresses provides utilities to work with addresses in cosmos-sdk based chains.
type Addresses struct {
	bech32Prefix string
}

// Derive derives an AddressRaw to its chain specific human-readable form.
func (a *Addresses) Derive(address AddressRaw) (AddressHuman, error) {
	return bech32.ConvertAndEncode(a.bech32Prefix, address)
}

// Decode decodes an AddressHuman representation into its AddressRaw format.
func (a *Addresses) Decode(address AddressHuman) (AddressRaw, error) {
	hrp, addressBytes, err := bech32.DecodeAndConvert(address)
	if err != nil {
		return nil, err
	}

	if a.bech32Prefix != hrp {
		return nil, fmt.Errorf("address prefix mismatcj expected: %s got: %s", a.bech32Prefix, hrp)
	}

	return addressBytes, nil
}
