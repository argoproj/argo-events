/*
Copyright 2026 The Argoproj Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package grpc

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

// scalarFieldTypes maps the GRPCSchemaField.Type string to the protobuf
// wire type. Only flat scalars are supported; nested messages and repeated
// fields are out of scope.
var scalarFieldTypes = map[string]descriptorpb.FieldDescriptorProto_Type{
	"string": descriptorpb.FieldDescriptorProto_TYPE_STRING,
	"int32":  descriptorpb.FieldDescriptorProto_TYPE_INT32,
	"int64":  descriptorpb.FieldDescriptorProto_TYPE_INT64,
	"uint32": descriptorpb.FieldDescriptorProto_TYPE_UINT32,
	"uint64": descriptorpb.FieldDescriptorProto_TYPE_UINT64,
	"float":  descriptorpb.FieldDescriptorProto_TYPE_FLOAT,
	"double": descriptorpb.FieldDescriptorProto_TYPE_DOUBLE,
	"bool":   descriptorpb.FieldDescriptorProto_TYPE_BOOL,
	"bytes":  descriptorpb.FieldDescriptorProto_TYPE_BYTES,
}

// emptyResponseDescriptor describes a message with zero fields. Since the
// GRPC trigger never decodes the response, every RPC reply is unmarshaled
// into a message built from this descriptor: protobuf's wire format allows
// unknown fields, so any real response message unmarshals cleanly without
// needing its actual schema.
var emptyResponseDescriptor = mustBuildEmptyResponseDescriptor()

func mustBuildEmptyResponseDescriptor() protoreflect.MessageDescriptor {
	fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String("argo-events/grpc_trigger_empty_response.proto"),
		Package: proto.String("argoevents.dynamic"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("EmptyResponse")},
		},
	}, protoregistry.GlobalFiles)
	if err != nil {
		panic(fmt.Errorf("failed to build the empty response descriptor: %w", err))
	}
	return fd.Messages().Get(0)
}

// buildRequestDescriptor builds a synthetic protobuf message descriptor
// for the gRPC trigger's request, from the user-supplied minimal schema.
func buildRequestDescriptor(schema []v1alpha1.GRPCSchemaField) (protoreflect.MessageDescriptor, error) {
	fields := make([]*descriptorpb.FieldDescriptorProto, 0, len(schema))
	seenNumbers := make(map[int32]bool, len(schema))

	for _, f := range schema {
		fieldType, ok := scalarFieldTypes[f.Type]
		if !ok {
			return nil, fmt.Errorf("unsupported schema field type %q for field %q", f.Type, f.Name)
		}
		if seenNumbers[f.Number] {
			return nil, fmt.Errorf("duplicate field number %d in schema", f.Number)
		}
		seenNumbers[f.Number] = true

		fields = append(fields, &descriptorpb.FieldDescriptorProto{
			Name:   proto.String(f.Name),
			Number: proto.Int32(f.Number),
			Type:   fieldType.Enum(),
			Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		})
	}

	fd, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String("argo-events/grpc_trigger_request.proto"),
		Package: proto.String("argoevents.dynamic"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:  proto.String("Request"),
				Field: fields,
			},
		},
	}, protoregistry.GlobalFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to build the request descriptor: %w", err)
	}
	return fd.Messages().Get(0), nil
}
