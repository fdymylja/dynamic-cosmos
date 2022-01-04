package dynamic

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func NewRegistry(remote RemoteRegistry) *Registry {
	return &Registry{
		remote:    remote,
		prefFiles: new(protoregistry.Files),
		prefTypes: new(protoregistry.Types),
	}
}

var (
	_ protodesc.Resolver = (*Registry)(nil)
)

type RemoteRegistry interface {
	ProtoFileByPath(ctx context.Context, path string) (*descriptorpb.FileDescriptorProto, error)
	ProtoFileContainingSymbol(ctx context.Context, name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error)
}

type Registry struct {
	remote RemoteRegistry

	prefFiles *protoregistry.Files
	prefTypes *protoregistry.Types
}

func (r Registry) FindFileByPath(s string) (protoreflect.FileDescriptor, error) {
	fd, err := r.prefFiles.FindFileByPath(s)
	if err == nil {
		return fd, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}

	// try fetch it from remote
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	dpb, err := r.remote.ProtoFileByPath(ctx, s)
	if err != nil {
		return nil, err
	}

	fd, err = protodesc.NewFile(dpb, r)
	if err != nil {
		return nil, err
	}

	err = r.prefFiles.RegisterFile(fd)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func (r Registry) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	desc, err := r.prefFiles.FindDescriptorByName(name)
	if err == nil {
		return desc, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}

	// try fetch from remote
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	dpb, err := r.remote.ProtoFileContainingSymbol(ctx, name)
	if err != nil {
		return nil, err
	}
	fd, err := protodesc.NewFile(dpb, r)
	if err != nil {
		return nil, err
	}

	err = r.prefFiles.RegisterFile(fd)
	if err != nil {
		return nil, err
	}

	return r.prefFiles.FindDescriptorByName(name)
}

var _ RemoteRegistry = (*grpcReflectionRemote)(nil)

// grpcReflectionRemote is a RemoteRegistry
// which uses grpc reflection to resolve files.
type grpcReflectionRemote struct {
	rpb grpc_reflection_v1alpha.ServerReflectionClient
}

func (g grpcReflectionRemote) ProtoFileByPath(ctx context.Context, path string) (*descriptorpb.FileDescriptorProto, error) {
	stream, err := g.rpb.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.CloseSend()

	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileByFilename{
			FileByFilename: path,
		}})
	if err != nil {
		return nil, err
	}

	recv, err := stream.Recv()
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

func (g grpcReflectionRemote) ProtoFileContainingSymbol(ctx context.Context, name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	stream, err := g.rpb.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.CloseSend()

	err = stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: string(name),
		},
	})

	if err != nil {
		return nil, err
	}

	recv, err := stream.Recv()
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
