package protoutil

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"strings"
)

// FullNameFromURL returns protoreflect.FullName from proto.Messages' typeURL
func FullNameFromURL(typeURL string) protoreflect.FullName {
	message := protoreflect.FullName(typeURL)
	if i := strings.LastIndexByte(typeURL, '/'); i >= 0 {
		message = message[i+len("/"):]
	}

	return message
}

func Merge(src, dst proto.Message, reset bool) {
	if src.ProtoReflect().Descriptor() != dst.ProtoReflect().Descriptor() && src.ProtoReflect().Descriptor().FullName() != dst.ProtoReflect().Descriptor().FullName() {
		panic(fmt.Sprintf("src and dst are not the same message: %s <-> %s", src.ProtoReflect().Descriptor().FullName(), dst.ProtoReflect().Descriptor().FullName()))
	}
	if reset {
		dst.ProtoReflect().Range(func(descriptor protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			dst.ProtoReflect().Clear(descriptor)
			return true
		})
	}

	src.ProtoReflect().Range(func(descriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		dst.ProtoReflect().Set(dst.ProtoReflect().Descriptor().Fields().ByName(descriptor.Name()), value)
		return true
	})

}
