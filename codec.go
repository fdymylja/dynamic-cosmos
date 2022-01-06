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

func (c *Codec) MarshalProto(m proto.Message) ([]byte, error) {
	return c.marshal.Marshal(m)
}

func (c *Codec) UnmarshalProto(b []byte, m proto.Message) error {
	return c.unmarshal.Unmarshal(b, m)
}

func (c *Codec) MarshalProtoJSON(m proto.Message) ([]byte, error) {
	return c.jsonMarshal.Marshal(m)
}

func (c *Codec) UnmarshalProtoJSON(b []byte, m proto.Message) error {
	return c.jsonUnmarshal.Unmarshal(b, m)
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
