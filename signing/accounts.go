package signing

import (
	"github.com/fdymylja/dynamic-cosmos/codec"
	"github.com/fdymylja/dynamic-cosmos/protoutil"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

var accountInfoProvider = map[protoreflect.FullName]func(codec *codec.Codec, any *anypb.Any) (pubKey *anypb.Any, sequence uint64, err error){
	"cosmos.auth.v1beta1.BaseAccount": func(codec *codec.Codec, any *anypb.Any) (pubKey *anypb.Any, sequence uint64, err error) {
		raw, err := anypb.UnmarshalNew(any, codec.ProtoOptions().Unmarshal)
		if err != nil {
			return nil, 0, err
		}

		sequence = raw.ProtoReflect().Get(raw.ProtoReflect().Descriptor().Fields().ByName("sequence")).Uint()
		pubKeyRaw := raw.ProtoReflect().Get(raw.ProtoReflect().Descriptor().Fields().ByName("pub_key")).Message()

		pubKey = new(anypb.Any)
		protoutil.Merge(pubKeyRaw.Interface(), pubKey, false)

		return
	},
}
