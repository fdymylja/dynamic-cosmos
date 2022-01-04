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
			Resolver: registry,
		},
		jsonMarshal: protojson.MarshalOptions{
			Multiline:       false,
			Indent:          "\t",
			AllowPartial:    false,
			UseProtoNames:   false,
			UseEnumNumbers:  false,
			EmitUnpopulated: false,
			Resolver:        registry,
		},
		jsonUnmarshal: protojson.UnmarshalOptions{
			AllowPartial:   false,
			DiscardUnknown: false,
			Resolver:       registry,
		},
	}
}
