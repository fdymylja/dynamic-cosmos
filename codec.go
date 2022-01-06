package dynamic

import (
	"fmt"
	"google.golang.org/grpc/encoding"
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

func (c *Codec) GRPCCodec() encoding.Codec {
	return &grpcCodec{
		m: c.marshal,
		u: c.unmarshal,
	}
}

var _ encoding.Codec = (*grpcCodec)(nil)

type grpcCodec struct {
	m proto.MarshalOptions
	u proto.UnmarshalOptions
}

func (g *grpcCodec) Marshal(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("dynamic cosmos grpc client can only work with proto.Message")
	}

	return g.m.Marshal(msg)
}

func (g *grpcCodec) Unmarshal(data []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("dynamic cosmos grpc client can only work with proto.Message")
	}

	return g.u.Unmarshal(data, msg)
}

func (g *grpcCodec) Name() string {
	return "dynamic-cosmos-codec"
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
