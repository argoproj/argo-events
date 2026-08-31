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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestBuildRequestDescriptor_FieldNumbersAndTypes(t *testing.T) {
	desc, err := buildRequestDescriptor([]v1alpha1.GRPCSchemaField{
		{Name: "message", Number: 1, Type: "string"},
		{Name: "count", Number: 5, Type: "int32"},
	})
	require.NoError(t, err)

	messageField := desc.Fields().ByName("message")
	require.NotNil(t, messageField)
	assert.EqualValues(t, 1, messageField.Number())
	assert.Equal(t, protoreflect.StringKind, messageField.Kind())

	countField := desc.Fields().ByName("count")
	require.NotNil(t, countField)
	assert.EqualValues(t, 5, countField.Number())
	assert.Equal(t, protoreflect.Int32Kind, countField.Kind())
}

func TestBuildRequestDescriptor_JSONRoundTrip(t *testing.T) {
	desc, err := buildRequestDescriptor([]v1alpha1.GRPCSchemaField{
		{Name: "message", Number: 1, Type: "string"},
		{Name: "count", Number: 2, Type: "int32"},
	})
	require.NoError(t, err)

	msg := dynamicpb.NewMessage(desc)
	err = protojson.Unmarshal([]byte(`{"message":"hi","count":3}`), msg)
	require.NoError(t, err)

	assert.Equal(t, "hi", msg.Get(desc.Fields().ByName("message")).String())
	assert.EqualValues(t, 3, msg.Get(desc.Fields().ByName("count")).Int())
}

func TestBuildRequestDescriptor_UnsupportedType(t *testing.T) {
	_, err := buildRequestDescriptor([]v1alpha1.GRPCSchemaField{
		{Name: "bad", Number: 1, Type: "map"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schema field type")
}

func TestBuildRequestDescriptor_DuplicateNumber(t *testing.T) {
	_, err := buildRequestDescriptor([]v1alpha1.GRPCSchemaField{
		{Name: "a", Number: 1, Type: "string"},
		{Name: "b", Number: 1, Type: "int32"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate field number")
}

func TestEmptyResponseDescriptor_AcceptsAnyBytesWithoutError(t *testing.T) {
	// A message built from the request descriptor, marshaled to bytes,
	// must unmarshal cleanly into the empty response descriptor (unknown
	// fields are legal in protobuf's wire format).
	reqDesc, err := buildRequestDescriptor([]v1alpha1.GRPCSchemaField{
		{Name: "message", Number: 1, Type: "string"},
	})
	require.NoError(t, err)

	reqMsg := dynamicpb.NewMessage(reqDesc)
	require.NoError(t, protojson.Unmarshal([]byte(`{"message":"hi"}`), reqMsg))

	wire, err := proto.Marshal(reqMsg)
	require.NoError(t, err)

	replyMsg := dynamicpb.NewMessage(emptyResponseDescriptor)
	assert.NoError(t, proto.Unmarshal(wire, replyMsg))
}
