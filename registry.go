package dynamic

import (
	"errors"

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
	ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error)
	ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error)
	Close() error
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

	dpb, err := r.remote.ProtoFileByPath(s)
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

	dpb, err := r.remote.ProtoFileContainingSymbol(name)
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

func (r *Registry) Save() (*descriptorpb.FileDescriptorSet, error) {
	set := &descriptorpb.FileDescriptorSet{File: make([]*descriptorpb.FileDescriptorProto, 0, r.prefFiles.NumFiles())}
	var err error
	r.prefFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		set.File = append(set.File, protodesc.ToFileDescriptorProto(fd))
		return true
	})

	return set, err
}
