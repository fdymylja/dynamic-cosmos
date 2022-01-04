package dynamic

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Codec struct {
	marshal       proto.MarshalOptions
	unmarshal     proto.UnmarshalOptions
	jsonMarshal   protojson.MarshalOptions
	jsonUnmarshal protojson.UnmarshalOptions
}

func NewCodec(registry *Registry) *Codec {
	return &Codec{
		marshal: proto.MarshalOptions{
			Deterministic: true,
		},
		unmarshal: proto.UnmarshalOptions{
			Resolver: nil,
		},
		jsonMarshal:   protojson.MarshalOptions{},
		jsonUnmarshal: protojson.UnmarshalOptions{},
	}
}
