package codec

import (
	"context"
	"log"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var _ ProtoFileRegistry = (*GRPCReflectionProtoFileRegistry)(nil)
var _ ProtoFileRegistry = (*CacheProtoFileRegistry)(nil)
var _ ProtoFileRegistry = (*MultiProtoFileRegistry)(nil)

func NewMultiProtoFileRegistry(remotes ...ProtoFileRegistry) *MultiProtoFileRegistry {
	return &MultiProtoFileRegistry{remotes: remotes}
}

type MultiProtoFileRegistry struct {
	remotes []ProtoFileRegistry
}

func (m MultiProtoFileRegistry) ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	for _, rem := range m.remotes {
		fdpb, err := rem.ProtoFileByPath(path)
		if err == nil {
			return fdpb, nil
		}

		log.Printf("remote %T didn't find path: %s", rem, path)
	}

	return nil, protoregistry.NotFound
}

func (m MultiProtoFileRegistry) ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	for _, rem := range m.remotes {
		fdpb, err := rem.ProtoFileContainingSymbol(name)
		if err == nil {
			return fdpb, nil
		}

		log.Printf("remote %T didn't find fullname: %s", rem, name)
	}

	return nil, protoregistry.NotFound
}

func (m MultiProtoFileRegistry) Close() error {
	for _, rem := range m.remotes {
		_ = rem.Close()
	}

	return nil
}

func NewCacheProtoFileRegistry(set *descriptorpb.FileDescriptorSet) *CacheProtoFileRegistry {
	return &CacheProtoFileRegistry{set: set}
}

type CacheProtoFileRegistry struct {
	set *descriptorpb.FileDescriptorSet
}

func (c CacheProtoFileRegistry) ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	for _, fdpb := range c.set.File {
		if fdpb.Name != nil && *fdpb.Name == path {
			return fdpb, nil
		}
	}

	return nil, protoregistry.NotFound
}

func (c CacheProtoFileRegistry) ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	for _, fdpb := range c.set.File {
		fdFullName := protoreflect.FullName(fdpb.GetPackage())
		if fdFullName == name {
			return fdpb, nil
		}
		// check messages
		for _, md := range fdpb.MessageType {
			found := findNameInDescriptorProto(name, fdFullName, md)
			if found {
				return fdpb, nil
			}
		}
		// check services
		for _, sd := range fdpb.Service {
			sdName := protoreflect.Name(sd.GetName())
			sdFullName := fdFullName.Append(sdName)
			if sdFullName == name {
				return fdpb, nil
			}
			// check methods inside services
			for _, md := range sd.Method {
				mdName := protoreflect.Name(md.GetName())
				mdFullName := sdFullName.Append(mdName)
				if mdFullName == name {
					return fdpb, nil
				}
			}
		}
		// check enums
		for _, ed := range fdpb.EnumType {
			found := findNameInEnum(name, fdFullName, ed)
			if found {
				return fdpb, nil
			}
		}
		// check extension
		for _, xd := range fdpb.Extension {
			xdFullName := fdFullName.Append(protoreflect.Name(xd.GetName()))
			if xdFullName == name {
				return fdpb, nil
			}
		}
	}

	return nil, protoregistry.NotFound
}

func (c CacheProtoFileRegistry) Close() error {
	return nil
}

func findNameInEnum(name, parent protoreflect.FullName, desc *descriptorpb.EnumDescriptorProto) bool {
	// check enum
	self := parent.Append(protoreflect.Name(desc.GetName()))
	if self == name {
		return true
	}

	// check values
	for _, value := range desc.Value {
		valueFullName := self.Append(protoreflect.Name(value.GetName()))
		if valueFullName == name {
			return true
		}
	}

	return false
}

func findNameInDescriptorProto(name, parent protoreflect.FullName, desc *descriptorpb.DescriptorProto) bool {
	// check self
	self := parent.Append(protoreflect.Name(desc.GetName()))
	if self == name {
		return true
	}
	// check oneofs
	for _, oneof := range desc.OneofDecl {
		oneofFullName := self.Append(protoreflect.Name(oneof.GetName()))
		if oneofFullName == name {
			return true
		}
	}
	// check fields
	for _, fd := range desc.Field {
		fdFullName := self.Append(protoreflect.Name(fd.GetName()))
		if fdFullName == name {
			return true
		}
	}
	// check nested enums
	for _, ed := range desc.EnumType {
		found := findNameInEnum(name, self, ed)
		if found {
			return true
		}
	}
	// check extensions
	for _, xd := range desc.Extension {
		xdFullName := self.Append(protoreflect.Name(xd.GetName()))
		if xdFullName == name {
			return true
		}
	}
	// check nested types
	for _, nt := range desc.NestedType {
		found := findNameInDescriptorProto(name, self, nt)
		if found {
			return true
		}
	}
	return false
}

func NewGRPCReflectionProtoFileRegistry(grpcEndpoint string) (*GRPCReflectionProtoFileRegistry, error) {
	conn, err := grpc.Dial(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &GRPCReflectionProtoFileRegistry{
		rpb:    grpc_reflection_v1alpha.NewServerReflectionClient(conn),
		once:   new(sync.Once),
		stream: nil,
	}, nil
}

// GRPCReflectionProtoFileRegistry is a ProtoFileRegistry
// which uses grpc reflection to resolve files.
type GRPCReflectionProtoFileRegistry struct {
	rpb    grpc_reflection_v1alpha.ServerReflectionClient
	once   *sync.Once
	stream grpc_reflection_v1alpha.ServerReflection_ServerReflectionInfoClient
}

func (g *GRPCReflectionProtoFileRegistry) ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	err := g.init()
	if err != nil {
		return nil, err
	}

	err = g.stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileByFilename{
			FileByFilename: path,
		}})
	if err != nil {
		return nil, err
	}

	recv, err := g.stream.Recv()
	if err != nil {
		return nil, err
	}

	resp := recv.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse)
	fdRawBytes := resp.FileDescriptorResponse.FileDescriptorProto[0]
	fdPb := &descriptorpb.FileDescriptorProto{}
	err = proto.Unmarshal(fdRawBytes, fdPb)
	if err != nil {
		return nil, err
	}

	return fdPb, nil
}

func (g *GRPCReflectionProtoFileRegistry) ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	err := g.init()
	if err != nil {
		return nil, err
	}

	err = g.stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: string(name),
		},
	})

	if err != nil {
		return nil, err
	}

	recv, err := g.stream.Recv()
	if err != nil {
		return nil, err
	}

	resp := recv.MessageResponse.(*grpc_reflection_v1alpha.ServerReflectionResponse_FileDescriptorResponse)
	fdRawBytes := resp.FileDescriptorResponse.FileDescriptorProto[0]
	fdPb := &descriptorpb.FileDescriptorProto{}
	err = proto.Unmarshal(fdRawBytes, fdPb)
	if err != nil {
		return nil, err
	}

	return fdPb, nil
}

func (g *GRPCReflectionProtoFileRegistry) init() (err error) {
	g.once.Do(func() {
		g.stream, err = g.rpb.ServerReflectionInfo(context.Background())
	})
	return err
}

func (g *GRPCReflectionProtoFileRegistry) Close() error {
	return g.stream.CloseSend()
}
