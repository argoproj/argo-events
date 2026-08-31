# gRPC Trigger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a new `GRPC` Sensor trigger type that invokes an arbitrary unary gRPC method on a target server, building the request message from a user-supplied minimal field schema instead of compiled client stubs.

**Architecture:** A new `pkg/sensors/triggers/grpc` package implements the existing `Trigger` interface (`GetTriggerType`/`FetchResource`/`ApplyResourceParameters`/`Execute`/`ApplyPolicy`), mirroring `pkg/sensors/triggers/http`. At `Execute` time it builds a synthetic protobuf `MessageDescriptor` from the trigger's `Schema` field (via `protodesc`), populates a `dynamicpb.Message` from the JSON payload (via `protojson`, built the same way every other trigger's `Payload` is built), and calls `grpc.ClientConn.Invoke` directly against the fully-qualified method path — no generated Go stubs needed for the target service. The response is discarded; only the returned gRPC status code is kept, reusing the existing generic `Trigger.Policy.Status.Allow` / `Trigger.RetryStrategy` mechanism that every other trigger type already relies on for success-checking and backoff.

**Tech Stack:** Go, `google.golang.org/grpc` v1.81.1, `google.golang.org/protobuf` v1.36.11 (both already in go.mod — no new dependency), Kubernetes CRD codegen tooling (`make codegen`).

## Global Constraints

- v1 supports unary RPCs only — no streaming.
- No new third-party dependency — only already-vendored `google.golang.org/grpc` and `google.golang.org/protobuf` packages.
- `Schema` fields are flat scalars only: `string`, `int32`, `int64`, `uint32`, `uint64`, `float`, `double`, `bool`, `bytes`. No nested messages, no repeated fields.
- Field `Number` is explicit and required in the schema (never inferred from position) — it must match the real field number in the target's `.proto`.
- The response body is never decoded — only the gRPC status code is used, via the existing shared `v1alpha1.Trigger.Policy.Status.Allow []int32` field (interpreted as gRPC status codes for this trigger type) and the existing shared `v1alpha1.Trigger.RetryStrategy` backoff. No new policy/backoff type is introduced.
- gRPC connections are cached and reused, keyed by `trigger.Template.Name`, exactly like `httpClients` and `customTriggerClients` already are.
- Follow existing per-trigger-type file/package conventions exactly (see `pkg/sensors/triggers/http/http.go` and `pkg/sensors/triggers/custom-trigger/custom-trigger.go`).
- Spec: `docs/superpowers/specs/2026-07-01-grpc-trigger-design.md`.

---

## Task 1: CRD API types

**Files:**
- Modify: `pkg/apis/events/v1alpha1/const.go:76-94` (add `TriggerTypeGRPC`)
- Modify: `pkg/apis/events/v1alpha1/sensor_types.go:290-342` (add `GRPC *GRPCTrigger` field to `TriggerTemplate`)
- Modify: `pkg/apis/events/v1alpha1/sensor_types.go` (add `GRPCTrigger` and `GRPCSchemaField` structs, placed after the `HTTPTrigger`/`SecureHeader`/`ValueFromSource` block that ends at line 449, i.e. right before `AWSLambdaTrigger` at line 451)
- Test: `pkg/apis/events/v1alpha1/sensor_types_test.go` (new file)

**Interfaces:**
- Produces: `v1alpha1.TriggerTypeGRPC` (a `TriggerType` constant), `v1alpha1.GRPCTrigger` struct with fields `URL string`, `Method string`, `Schema []GRPCSchemaField`, `Payload []TriggerParameter`, `TLS *TLSConfig`, `Insecure bool`, `Timeout int64`, `Parameters []TriggerParameter`. `v1alpha1.GRPCSchemaField` struct with fields `Name string`, `Number int32`, `Type string`. `v1alpha1.TriggerTemplate.GRPC *GRPCTrigger`.

- [ ] **Step 1: Write the failing test**

Create `pkg/apis/events/v1alpha1/sensor_types_test.go`:

```go
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
package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGRPCTriggerDeepCopy(t *testing.T) {
	original := &GRPCTrigger{
		URL:    "localhost:9000",
		Method: "/helloworld.Greeter/SayHello",
		Schema: []GRPCSchemaField{
			{Name: "message", Number: 1, Type: "string"},
		},
		Payload: []TriggerParameter{
			{Dest: "message"},
		},
		Insecure: true,
		Timeout:  10,
	}

	copied := original.DeepCopy()
	assert.Equal(t, original, copied)

	copied.Schema[0].Name = "changed"
	assert.Equal(t, "message", original.Schema[0].Name, "DeepCopy must not alias the Schema slice")
}

func TestTriggerTemplateGRPCField(t *testing.T) {
	template := &TriggerTemplate{
		Name: "test",
		GRPC: &GRPCTrigger{URL: "localhost:9000", Method: "/pkg.Svc/Method"},
	}
	assert.Equal(t, "localhost:9000", template.GRPC.URL)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/apis/events/v1alpha1/... -run 'TestGRPCTriggerDeepCopy|TestTriggerTemplateGRPCField' -v`
Expected: FAIL — compile error, `GRPCTrigger`/`GRPCSchemaField`/`TriggerTemplate.GRPC`/`DeepCopy` undefined.

- [ ] **Step 3: Add the constant**

In `pkg/apis/events/v1alpha1/const.go`, in the `TriggerType` var block (lines 79-94), add a new line after `TriggerTypeEmail`:

```go
	TriggerTypeEmail           TriggerType = "Email"
	TriggerTypeGRPC            TriggerType = "GRPC"
)
```

- [ ] **Step 4: Add the `GRPC` field to `TriggerTemplate`**

In `pkg/apis/events/v1alpha1/sensor_types.go`, inside `TriggerTemplate` (ends at line 342 with `}`), add before the closing brace:

```go
	// Email refers to the trigger designed to send an email notification
	// +optional
	Email *EmailTrigger `json:"email,omitempty" protobuf:"bytes,17,opt,name=email"`
	// GRPC refers to the trigger designed to invoke an arbitrary unary gRPC method.
	// +optional
	GRPC *GRPCTrigger `json:"grpc,omitempty" protobuf:"bytes,18,opt,name=grpc"`
}
```

- [ ] **Step 5: Add the `GRPCTrigger` and `GRPCSchemaField` structs**

In `pkg/apis/events/v1alpha1/sensor_types.go`, immediately after the `ValueFromSource` struct (ends at line 449 with `}`) and before `// AWSLambdaTrigger refers to...` (line 451), insert:

```go
// GRPCTrigger is the trigger to invoke an arbitrary unary gRPC method on a
// target server. Since no compiled client stubs exist for the target
// service, the request message shape is described by Schema.
type GRPCTrigger struct {
	// URL is the target gRPC server address (host:port).
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
	// Method is the fully-qualified RPC method path,
	// e.g. "/helloworld.Greeter/SayHello".
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`
	// Schema describes the request message's fields. Number MUST match the
	// field number in the target's real .proto definition, or the message
	// will be silently misinterpreted on the wire.
	// +optional
	Schema []GRPCSchemaField `json:"schema,omitempty" protobuf:"bytes,3,rep,name=schema"`
	// Payload is the list of key-value extracted from an event payload to
	// construct the gRPC request message.
	// +optional
	Payload []TriggerParameter `json:"payload,omitempty" protobuf:"bytes,4,rep,name=payload"`
	// TLS configuration for the gRPC connection. Mutually exclusive with Insecure.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty" protobuf:"bytes,5,opt,name=tls"`
	// Insecure disables TLS entirely (plaintext connection).
	// +optional
	Insecure bool `json:"insecure,omitempty" protobuf:"varint,6,opt,name=insecure"`
	// Timeout for the RPC call, in seconds. Defaults to 10.
	// +optional
	Timeout int64 `json:"timeout,omitempty" protobuf:"varint,7,opt,name=timeout"`
	// Parameters is the list of key-value extracted from event's payload
	// that are applied to the gRPC trigger resource.
	// +optional
	Parameters []TriggerParameter `json:"parameters,omitempty" protobuf:"bytes,8,rep,name=parameters"`
}

// GRPCSchemaField describes a single field of the gRPC request message.
type GRPCSchemaField struct {
	// Name is the proto/JSON field name.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Number is the protobuf field number, taken from the target's .proto.
	Number int32 `json:"number" protobuf:"varint,2,opt,name=number"`
	// Type is the scalar field type: string, int32, int64, uint32, uint64,
	// float, double, bool, or bytes.
	Type string `json:"type" protobuf:"bytes,3,opt,name=type"`
}

```

- [ ] **Step 6: Run codegen**

Run: `make codegen`

This regenerates `zz_generated.deepcopy.go` (adds `DeepCopy`/`DeepCopyInto` for `GRPCTrigger`/`GRPCSchemaField` and updates `TriggerTemplate`'s), `generated.pb.go` (protobuf marshal/unmarshal), the CRD manifests under `manifests/`, `api/openapi-spec/swagger.json`, `api/jsonschema/schema.json`, and `docs/APIs.md`.

Prerequisite: `protoc` must be installed and match the version used elsewhere in this repo's tooling (check `hack/generate-proto.sh`). If `make codegen` fails because `protoc` (or another codegen tool) isn't installed, install it first (e.g. `brew install protobuf` on macOS, or your platform's package manager) and re-run — do not hand-write `generated.pb.go`.

Expected: command exits 0. Confirm with `git status --porcelain` that at minimum `pkg/apis/events/v1alpha1/zz_generated.deepcopy.go` and `pkg/apis/events/v1alpha1/generated.pb.go` show as modified.

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./pkg/apis/events/v1alpha1/... -run 'TestGRPCTriggerDeepCopy|TestTriggerTemplateGRPCField' -v`
Expected: PASS

- [ ] **Step 8: Build the whole repo to catch any downstream break**

Run: `go build ./...`
Expected: exits 0, no errors.

- [ ] **Step 9: Commit**

```bash
git add pkg/apis/events/v1alpha1/const.go pkg/apis/events/v1alpha1/sensor_types.go pkg/apis/events/v1alpha1/sensor_types_test.go pkg/apis/events/v1alpha1/zz_generated.deepcopy.go pkg/apis/events/v1alpha1/generated.pb.go pkg/client api/openapi-spec/swagger.json api/jsonschema/schema.json docs/APIs.md manifests
git commit -m "feat(sensors): add GRPC trigger CRD types"
```

---

## Task 2: Request-message descriptor builder

**Files:**
- Create: `pkg/sensors/triggers/grpc/schema.go`
- Test: `pkg/sensors/triggers/grpc/schema_test.go`

**Interfaces:**
- Consumes: `v1alpha1.GRPCSchemaField{Name string, Number int32, Type string}` (from Task 1).
- Produces: `buildRequestDescriptor(schema []v1alpha1.GRPCSchemaField) (protoreflect.MessageDescriptor, error)` and `emptyResponseDescriptor protoreflect.MessageDescriptor` (a package-level var) — both consumed by Task 3/4's `grpc.go`.

- [ ] **Step 1: Write the failing test**

Create `pkg/sensors/triggers/grpc/schema_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sensors/triggers/grpc/... -v`
Expected: FAIL — package `grpc` doesn't exist yet / `buildRequestDescriptor` undefined.

- [ ] **Step 3: Write the implementation**

Create `pkg/sensors/triggers/grpc/schema.go`:

```go
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
```

- [ ] **Step 4: Fix the test's missing import**

In `schema_test.go`, add `"google.golang.org/protobuf/proto"` to the import block (used by `proto.Marshal`/`proto.Unmarshal` in the last test).

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./pkg/sensors/triggers/grpc/... -v`
Expected: PASS (all 5 tests)

- [ ] **Step 6: Commit**

```bash
git add pkg/sensors/triggers/grpc/schema.go pkg/sensors/triggers/grpc/schema_test.go
git commit -m "feat(sensors): add gRPC trigger request descriptor builder"
```

---

## Task 3: GRPCTrigger struct, connection management, FetchResource/ApplyResourceParameters

**Files:**
- Create: `pkg/sensors/triggers/grpc/grpc.go`
- Test: `pkg/sensors/triggers/grpc/grpc_test.go`

**Interfaces:**
- Consumes: `buildRequestDescriptor`, `emptyResponseDescriptor` (Task 2). `v1alpha1.GRPCTrigger` (Task 1). `sharedutil.StringKeyedMap[*grpc.ClientConn]`, `sharedutil.GetTLSConfig(*v1alpha1.TLSConfig) (*tls.Config, error)` (existing, `pkg/shared/util`). `triggers.ApplyParams([]byte, []v1alpha1.TriggerParameter, map[string]*v1alpha1.Event) ([]byte, error)` (existing, `pkg/sensors/triggers/params.go:114`).
- Produces: `type GRPCTrigger struct { Conn *grpc.ClientConn; Sensor *v1alpha1.Sensor; Trigger *v1alpha1.Trigger; Logger *zap.SugaredLogger }`, `func NewGRPCTrigger(grpcClients sharedutil.StringKeyedMap[*grpc.ClientConn], sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) (*GRPCTrigger, error)`, `(*GRPCTrigger).GetTriggerType`, `(*GRPCTrigger).FetchResource`, `(*GRPCTrigger).ApplyResourceParameters` — consumed by Task 4 (same file/struct, added to in that task) and Task 6 (wiring).

- [ ] **Step 1: Write the failing test**

Create `pkg/sensors/triggers/grpc/grpc_test.go`:

```go
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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

func getFakeSensorAndTrigger() (*v1alpha1.Sensor, *v1alpha1.Trigger) {
	trigger := &v1alpha1.Trigger{
		Template: &v1alpha1.TriggerTemplate{
			Name: "fake-grpc-trigger",
			GRPC: &v1alpha1.GRPCTrigger{
				URL:      "localhost:0",
				Method:   "/test.EchoService/Echo",
				Insecure: true,
				Schema: []v1alpha1.GRPCSchemaField{
					{Name: "message", Number: 1, Type: "string"},
				},
				Payload: []v1alpha1.TriggerParameter{
					{Dest: "message"},
				},
			},
		},
	}
	sensor := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-sensor", Namespace: "fake"},
		Spec:       v1alpha1.SensorSpec{Triggers: []v1alpha1.Trigger{*trigger}},
	}
	return sensor, trigger
}

func TestNewGRPCTrigger_InsecureConnectionAndCaching(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	logger := logging.NewArgoEventsLogger()

	gt, err := NewGRPCTrigger(clients, sensor, trigger, logger)
	require.NoError(t, err)
	require.NotNil(t, gt.Conn)

	cachedConn, ok := clients.Load(trigger.Template.Name)
	require.True(t, ok)
	assert.Same(t, gt.Conn, cachedConn)

	// Calling it again with the same trigger name must reuse the cached conn.
	gt2, err := NewGRPCTrigger(clients, sensor, trigger, logger)
	require.NoError(t, err)
	assert.Same(t, gt.Conn, gt2.Conn)
}

func TestNewGRPCTrigger_RequiresTLSOrInsecure(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.Insecure = false
	trigger.Template.GRPC.TLS = nil
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()

	_, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either tls or insecure")
}

func TestGRPCTrigger_GetTriggerType(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)
	assert.Equal(t, v1alpha1.TriggerTypeGRPC, gt.GetTriggerType())
}

func TestGRPCTrigger_FetchResource(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	res, err := gt.FetchResource(context.TODO())
	require.NoError(t, err)
	fetched, ok := res.(*v1alpha1.GRPCTrigger)
	require.True(t, ok)
	assert.Equal(t, "localhost:0", fetched.URL)
}

func TestGRPCTrigger_ApplyResourceParameters(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	defaultURL := "localhost:9999"
	trigger.Template.GRPC.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dep",
				DataKey:        "url",
				Value:          &defaultURL,
			},
			Dest: "url",
		},
	}

	events := map[string]*v1alpha1.Event{
		"fake-dep": {
			Context: &v1alpha1.EventContext{ID: "1"},
			Data:    []byte(`{"url":"localhost:8888"}`),
		},
	}

	res, err := gt.ApplyResourceParameters(events, trigger.Template.GRPC)
	require.NoError(t, err)
	updated, ok := res.(*v1alpha1.GRPCTrigger)
	require.True(t, ok)
	assert.Equal(t, "localhost:8888", updated.URL)
}

func TestNewGRPCTrigger_TLSPath(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.Insecure = false
	// TLSConfig.Enabled with no cert secrets set still yields a valid empty
	// tls.Config (see sharedutil.GetTLSConfig) — enough to exercise the TLS
	// branch of NewGRPCTrigger without needing real certificates.
	trigger.Template.GRPC.TLS = &v1alpha1.TLSConfig{Enabled: true}

	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)
	assert.NotNil(t, gt.Conn)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sensors/triggers/grpc/... -v`
Expected: FAIL — `GRPCTrigger`/`NewGRPCTrigger` undefined in package `grpc`.

- [ ] **Step 3: Write the implementation**

Create `pkg/sensors/triggers/grpc/grpc.go`:

```go
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
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// GRPCTrigger describes the trigger to invoke an arbitrary unary gRPC method
type GRPCTrigger struct {
	// Conn is the gRPC client connection.
	Conn *grpc.ClientConn
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Trigger reference
	Trigger *v1alpha1.Trigger
	// Logger to log stuff
	Logger *zap.SugaredLogger
}

// NewGRPCTrigger returns a new GRPC trigger
func NewGRPCTrigger(grpcClients sharedutil.StringKeyedMap[*grpc.ClientConn], sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger, logger *zap.SugaredLogger) (*GRPCTrigger, error) {
	log := logger.With(logging.LabelTriggerType, v1alpha1.TriggerTypeGRPC)
	grpcTrigger := trigger.Template.GRPC

	if conn, ok := grpcClients.Load(trigger.Template.Name); ok {
		return &GRPCTrigger{Conn: conn, Sensor: sensor, Trigger: trigger, Logger: log}, nil
	}

	var creds credentials.TransportCredentials
	switch {
	case grpcTrigger.Insecure:
		creds = insecure.NewCredentials()
	case grpcTrigger.TLS != nil:
		tlsConfig, err := sharedutil.GetTLSConfig(grpcTrigger.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		creds = credentials.NewTLS(tlsConfig)
	default:
		return nil, fmt.Errorf("either tls or insecure must be configured for the gRPC trigger")
	}

	conn, err := grpc.NewClient(grpcTrigger.URL, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the gRPC server, %w", err)
	}

	grpcClients.Store(trigger.Template.Name, conn)

	return &GRPCTrigger{Conn: conn, Sensor: sensor, Trigger: trigger, Logger: log}, nil
}

// GetTriggerType returns the type of the trigger
func (t *GRPCTrigger) GetTriggerType() v1alpha1.TriggerType {
	return v1alpha1.TriggerTypeGRPC
}

// FetchResource fetches the trigger. As the GRPC trigger simply invokes a
// gRPC method, there is no need to fetch any resource from an external source.
func (t *GRPCTrigger) FetchResource(ctx context.Context) (interface{}, error) {
	return t.Trigger.Template.GRPC, nil
}

// ApplyResourceParameters applies parameters to the trigger resource
func (t *GRPCTrigger) ApplyResourceParameters(events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	fetchedResource, ok := resource.(*v1alpha1.GRPCTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the fetched trigger resource")
	}

	resourceBytes, err := json.Marshal(fetchedResource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the grpc trigger resource, %w", err)
	}
	parameters := fetchedResource.Parameters
	if parameters != nil {
		updatedResourceBytes, err := triggers.ApplyParams(resourceBytes, parameters, events)
		if err != nil {
			return nil, err
		}
		var gt *v1alpha1.GRPCTrigger
		if err := json.Unmarshal(updatedResourceBytes, &gt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the updated grpc trigger resource after applying resource parameters, %w", err)
		}
		return gt, nil
	}
	return resource, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sensors/triggers/grpc/... -v`
Expected: PASS (all 6 tests). Note `TestNewGRPCTrigger_InsecureConnectionAndCaching` and `TestNewGRPCTrigger_TLSPath` both succeed without a live server — `grpc.NewClient` connects lazily and doesn't dial until the first RPC.

- [ ] **Step 5: Commit**

```bash
git add pkg/sensors/triggers/grpc/grpc.go pkg/sensors/triggers/grpc/grpc_test.go
git commit -m "feat(sensors): add GRPCTrigger connection management and resource fetch"
```

---

## Task 4: Execute and ApplyPolicy (full RPC round-trip)

**Files:**
- Modify: `pkg/sensors/triggers/grpc/grpc.go` (add `Execute`, `ApplyPolicy`, and a `defaultGRPCTimeout` const)
- Modify: `pkg/sensors/triggers/grpc/grpc_test.go` (add an in-process test gRPC server + round-trip tests)

**Interfaces:**
- Consumes: `buildRequestDescriptor`, `emptyResponseDescriptor` (Task 2). `triggers.ConstructPayload(map[string]*v1alpha1.Event, []v1alpha1.TriggerParameter) ([]byte, error)` (existing, `pkg/sensors/triggers/params.go:40`). `pkg/sensors/policy.NewStatusPolicy(status int, statuses []int) *StatusPolicy` and `(*StatusPolicy).ApplyPolicy(ctx) error` (existing, `pkg/sensors/policy/status.go`).
- Produces: `(*GRPCTrigger).Execute(ctx, events, resource) (interface{}, error)` returning a `*status.Status` on a completed call (nil error), or `(nil, err)` only when the call could not even be attempted (bad schema, malformed payload). `(*GRPCTrigger).ApplyPolicy(ctx, resource) error` — both consumed by Task 6's wiring and by the Sensor's generic trigger-execution loop (`pkg/sensors/listener.go`), unchanged by this plan.

- [ ] **Step 1: Write the failing test**

Append to `pkg/sensors/triggers/grpc/grpc_test.go` (add these imports to the existing import block: `"net"`, `"google.golang.org/grpc/codes"`, `"google.golang.org/grpc/status"`, `"google.golang.org/protobuf/types/dynamicpb"`):

```go
// newTestEchoServer starts an in-process gRPC server exposing
// "/test.EchoService/Echo" without any compiled stubs: the handler decodes
// the raw request into a dynamic message built from the same schema the
// client uses, and returns replyErr (nil for success).
func newTestEchoServer(t *testing.T, schema []v1alpha1.GRPCSchemaField, replyErr error) (addr string, capturedMessage func() *dynamicpb.Message, stop func()) {
	t.Helper()

	desc, err := buildRequestDescriptor(schema)
	require.NoError(t, err)

	var captured *dynamicpb.Message

	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "test.EchoService",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Echo",
				Handler: func(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
					req := dynamicpb.NewMessage(desc)
					if err := dec(req); err != nil {
						return nil, err
					}
					captured = req
					if replyErr != nil {
						return nil, replyErr
					}
					return dynamicpb.NewMessage(emptyResponseDescriptor), nil
				},
			},
		},
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	server.RegisterService(serviceDesc, struct{}{})

	go func() {
		_ = server.Serve(lis)
	}()

	return lis.Addr().String(), func() *dynamicpb.Message { return captured }, server.Stop
}

func TestGRPCTrigger_Execute_Success(t *testing.T) {
	schema := []v1alpha1.GRPCSchemaField{
		{Name: "message", Number: 1, Type: "string"},
	}
	addr, capturedMessage, stop := newTestEchoServer(t, schema, nil)
	defer stop()

	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.URL = addr
	trigger.Template.GRPC.Schema = schema

	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	events := map[string]*v1alpha1.Event{
		"fake-dep": {
			Context: &v1alpha1.EventContext{ID: "1"},
			Data:    []byte(`{"message":"hello"}`),
		},
	}
	trigger.Template.GRPC.Payload = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{DependencyName: "fake-dep", DataKey: "message"},
			Dest: "message",
		},
	}

	result, err := gt.Execute(context.TODO(), events, trigger.Template.GRPC)
	require.NoError(t, err)

	st, ok := result.(*status.Status)
	require.True(t, ok)
	assert.Equal(t, codes.OK, st.Code())

	msg := capturedMessage()
	require.NotNil(t, msg)
	assert.Equal(t, "hello", msg.Get(msg.Descriptor().Fields().ByName("message")).String())
}

func TestGRPCTrigger_Execute_NonOKStatusIsNotAGoError(t *testing.T) {
	schema := []v1alpha1.GRPCSchemaField{
		{Name: "message", Number: 1, Type: "string"},
	}
	addr, _, stop := newTestEchoServer(t, schema, status.Error(codes.NotFound, "no such thing"))
	defer stop()

	sensor, trigger := getFakeSensorAndTrigger()
	trigger.Template.GRPC.URL = addr
	trigger.Template.GRPC.Schema = schema
	trigger.Template.GRPC.Payload = []v1alpha1.TriggerParameter{{Dest: "message"}}

	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	result, err := gt.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Template.GRPC)
	require.NoError(t, err, "a non-OK gRPC status must not surface as a Go error from Execute")

	st, ok := result.(*status.Status)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestGRPCTrigger_ApplyPolicy(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	gt.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{int32(codes.OK)}},
	}
	assert.NoError(t, gt.ApplyPolicy(context.TODO(), status.New(codes.OK, "")))

	gt.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{int32(codes.OK)}},
	}
	err = gt.ApplyPolicy(context.TODO(), status.New(codes.NotFound, "no such thing"))
	assert.Error(t, err)
}

func TestGRPCTrigger_ApplyPolicy_NoPolicyMeansNoCheck(t *testing.T) {
	sensor, trigger := getFakeSensorAndTrigger()
	clients := sharedutil.NewStringKeyedMap[*grpc.ClientConn]()
	gt, err := NewGRPCTrigger(clients, sensor, trigger, logging.NewArgoEventsLogger())
	require.NoError(t, err)

	assert.NoError(t, gt.ApplyPolicy(context.TODO(), status.New(codes.Internal, "boom")))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sensors/triggers/grpc/... -v`
Expected: FAIL — `(*GRPCTrigger).Execute` / `ApplyPolicy` undefined.

- [ ] **Step 3: Write the implementation**

In `pkg/sensors/triggers/grpc/grpc.go`, add to the import block:

```go
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/argoproj/argo-events/pkg/sensors/policy"
```

Add this const near the top of the file (after the imports, before the `GRPCTrigger` struct):

```go
const defaultGRPCTimeout = 10 * time.Second
```

Append these two methods at the end of the file:

```go
// Execute executes the trigger. The response body is never decoded — the
// returned value is always a *status.Status, even when the RPC completes
// with a non-OK code, mirroring how HTTPTrigger.Execute returns the raw
// *http.Response for non-2xx codes. A non-nil error here means the call
// could not even be attempted (bad schema, malformed payload).
func (t *GRPCTrigger) Execute(ctx context.Context, events map[string]*v1alpha1.Event, resource interface{}) (interface{}, error) {
	trigger, ok := resource.(*v1alpha1.GRPCTrigger)
	if !ok {
		return nil, fmt.Errorf("failed to interpret the trigger resource")
	}

	desc, err := buildRequestDescriptor(trigger.Schema)
	if err != nil {
		return nil, err
	}

	var payload []byte
	if trigger.Payload != nil {
		payload, err = triggers.ConstructPayload(events, trigger.Payload)
		if err != nil {
			return nil, err
		}
	}

	msg := dynamicpb.NewMessage(desc)
	if len(payload) > 0 {
		if err := protojson.Unmarshal(payload, msg); err != nil {
			return nil, fmt.Errorf("failed to construct the gRPC request message, %w", err)
		}
	}

	timeout := defaultGRPCTimeout
	if trigger.Timeout > 0 {
		timeout = time.Duration(trigger.Timeout) * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	t.Logger.Infow("invoking gRPC method...", zap.Any("url", trigger.URL), zap.Any("method", trigger.Method))

	reply := dynamicpb.NewMessage(emptyResponseDescriptor)
	invokeErr := t.Conn.Invoke(callCtx, trigger.Method, msg, reply)
	return status.Convert(invokeErr), nil
}

// ApplyPolicy applies policy on the trigger, reusing the same generic
// status-code allow-list mechanism every other trigger type uses
// (v1alpha1.Trigger.Policy.Status.Allow), interpreting Allow's values as
// gRPC status codes instead of HTTP status codes.
func (t *GRPCTrigger) ApplyPolicy(ctx context.Context, resource interface{}) error {
	if t.Trigger.Policy == nil || t.Trigger.Policy.Status == nil || t.Trigger.Policy.Status.Allow == nil {
		return nil
	}
	st, ok := resource.(*status.Status)
	if !ok {
		return fmt.Errorf("failed to interpret the trigger execution response")
	}

	p := policy.NewStatusPolicy(int(st.Code()), t.Trigger.Policy.Status.GetAllow())
	return p.ApplyPolicy(ctx)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/sensors/triggers/grpc/... -v`
Expected: PASS (all tests in the package)

- [ ] **Step 5: Run go vet and the full package build**

Run: `go vet ./pkg/sensors/triggers/grpc/... && go build ./...`
Expected: both exit 0.

- [ ] **Step 6: Commit**

```bash
git add pkg/sensors/triggers/grpc/grpc.go pkg/sensors/triggers/grpc/grpc_test.go
git commit -m "feat(sensors): add GRPCTrigger Execute and ApplyPolicy"
```

---

## Task 5: CRD-level validation

**Files:**
- Modify: `pkg/reconciler/sensor/validate.go` (add `validateGRPCTrigger`, wire it into `validateTriggerTemplate` and `validateTriggerPolicy`)
- Test: `pkg/reconciler/sensor/validate_test.go`

**Interfaces:**
- Consumes: `v1alpha1.GRPCTrigger`, `v1alpha1.GRPCSchemaField` (Task 1). `scalarFieldTypes` is intentionally NOT reused across packages (it lives in the `grpc` trigger package as an implementation detail) — this task defines its own small set of valid type names for validation purposes, kept in sync manually since both lists are short and stable.
- Produces: `validateGRPCTrigger(trigger *v1alpha1.GRPCTrigger) error`, used only within `pkg/reconciler/sensor/validate.go`.

- [ ] **Step 1: Write the failing test**

Append to `pkg/reconciler/sensor/validate_test.go`:

```go
func TestValidateGRPCTrigger(t *testing.T) {
	t.Run("valid trigger", func(t *testing.T) {
		trigger := &v1alpha1.GRPCTrigger{
			URL:      "localhost:9000",
			Method:   "/helloworld.Greeter/SayHello",
			Insecure: true,
			Schema: []v1alpha1.GRPCSchemaField{
				{Name: "message", Number: 1, Type: "string"},
			},
			Payload: []v1alpha1.TriggerParameter{
				{
					Src:  &v1alpha1.TriggerParameterSource{DependencyName: "dep"},
					Dest: "message",
				},
			},
		}
		assert.NoError(t, validateGRPCTrigger(trigger))
	})

	t.Run("nil trigger", func(t *testing.T) {
		assert.Error(t, validateGRPCTrigger(nil))
	})

	t.Run("missing url", func(t *testing.T) {
		trigger := &v1alpha1.GRPCTrigger{Method: "/pkg.Svc/Method", Insecure: true}
		err := validateGRPCTrigger(trigger)
		assert.ErrorContains(t, err, "server URL")
	})

	t.Run("missing method", func(t *testing.T) {
		trigger := &v1alpha1.GRPCTrigger{URL: "localhost:9000", Insecure: true}
		err := validateGRPCTrigger(trigger)
		assert.ErrorContains(t, err, "method")
	})

	t.Run("neither tls nor insecure", func(t *testing.T) {
		trigger := &v1alpha1.GRPCTrigger{URL: "localhost:9000", Method: "/pkg.Svc/Method"}
		err := validateGRPCTrigger(trigger)
		assert.ErrorContains(t, err, "tls")
	})

	t.Run("unsupported schema field type", func(t *testing.T) {
		trigger := &v1alpha1.GRPCTrigger{
			URL: "localhost:9000", Method: "/pkg.Svc/Method", Insecure: true,
			Schema: []v1alpha1.GRPCSchemaField{{Name: "x", Number: 1, Type: "map"}},
		}
		err := validateGRPCTrigger(trigger)
		assert.ErrorContains(t, err, "unsupported")
	})

	t.Run("non-positive field number", func(t *testing.T) {
		trigger := &v1alpha1.GRPCTrigger{
			URL: "localhost:9000", Method: "/pkg.Svc/Method", Insecure: true,
			Schema: []v1alpha1.GRPCSchemaField{{Name: "x", Number: 0, Type: "string"}},
		}
		err := validateGRPCTrigger(trigger)
		assert.ErrorContains(t, err, "field number")
	})

	t.Run("duplicate field number", func(t *testing.T) {
		trigger := &v1alpha1.GRPCTrigger{
			URL: "localhost:9000", Method: "/pkg.Svc/Method", Insecure: true,
			Schema: []v1alpha1.GRPCSchemaField{
				{Name: "x", Number: 1, Type: "string"},
				{Name: "y", Number: 1, Type: "int32"},
			},
		}
		err := validateGRPCTrigger(trigger)
		assert.ErrorContains(t, err, "duplicate")
	})
}

func TestValidateTriggerPolicy_GRPC(t *testing.T) {
	trigger := &v1alpha1.Trigger{
		Template: &v1alpha1.TriggerTemplate{
			Name: "grpc-trigger",
			GRPC: &v1alpha1.GRPCTrigger{URL: "localhost:9000", Method: "/pkg.Svc/Method", Insecure: true},
		},
		Policy: &v1alpha1.TriggerPolicy{
			Status: &v1alpha1.StatusPolicy{Allow: []int32{0}},
		},
	}
	assert.NoError(t, validateTriggerPolicy(trigger))

	trigger.Policy.Status.Allow = nil
	assert.Error(t, validateTriggerPolicy(trigger))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/reconciler/sensor/... -run 'TestValidateGRPCTrigger|TestValidateTriggerPolicy_GRPC' -v`
Expected: FAIL — `validateGRPCTrigger` undefined.

- [ ] **Step 3: Write the implementation**

In `pkg/reconciler/sensor/validate.go`, add the dispatch in `validateTriggerTemplate` (after the `template.Email != nil` block, before the closing `return nil` at line 183):

```go
	if template.Email != nil {
		if err := validateEmailTrigger(template.Email); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	if template.GRPC != nil {
		if err := validateGRPCTrigger(template.GRPC); err != nil {
			return fmt.Errorf("template %s is invalid, %w", template.Name, err)
		}
	}
	return nil
}
```

Add the dispatch in `validateTriggerPolicy` (after the `trigger.Template.AWSLambda != nil` block, before its closing `return nil`):

```go
	if trigger.Template.AWSLambda != nil {
		return validateStatusPolicy(trigger.Policy.Status)
	}
	if trigger.Template.GRPC != nil {
		return validateStatusPolicy(trigger.Policy.Status)
	}
	return nil
}
```

Add the new function, right after `validateEmailTrigger` (or any other convenient spot among the other `validateXTrigger` functions):

```go
// grpcSchemaFieldTypes is the set of scalar types the GRPC trigger's
// schema accepts. Kept in sync with scalarFieldTypes in
// pkg/sensors/triggers/grpc/schema.go.
var grpcSchemaFieldTypes = map[string]bool{
	"string": true,
	"int32":  true,
	"int64":  true,
	"uint32": true,
	"uint64": true,
	"float":  true,
	"double": true,
	"bool":   true,
	"bytes":  true,
}

// validateGRPCTrigger validates the GRPC trigger
func validateGRPCTrigger(trigger *v1alpha1.GRPCTrigger) error {
	if trigger == nil {
		return fmt.Errorf("GRPC trigger can't be nil")
	}
	if trigger.URL == "" {
		return fmt.Errorf("server URL is not specified")
	}
	if trigger.Method == "" {
		return fmt.Errorf("gRPC method is not specified")
	}
	if !trigger.Insecure && trigger.TLS == nil {
		return fmt.Errorf("either tls or insecure must be configured for the gRPC trigger")
	}

	seenNumbers := make(map[int32]bool, len(trigger.Schema))
	for _, f := range trigger.Schema {
		if !grpcSchemaFieldTypes[f.Type] {
			return fmt.Errorf("unsupported schema field type %q for field %q", f.Type, f.Name)
		}
		if f.Number <= 0 {
			return fmt.Errorf("schema field %q has an invalid field number %d, must be positive", f.Name, f.Number)
		}
		if seenNumbers[f.Number] {
			return fmt.Errorf("duplicate field number %d in schema", f.Number)
		}
		seenNumbers[f.Number] = true
	}

	if trigger.Parameters != nil {
		for i, parameter := range trigger.Parameters {
			if err := validateTriggerParameter(&parameter); err != nil {
				return fmt.Errorf("resource parameter index: %d. err: %w", i, err)
			}
		}
	}
	if trigger.Payload != nil {
		for i, p := range trigger.Payload {
			if err := validateTriggerParameter(&p); err != nil {
				return fmt.Errorf("payload index: %d. err: %w", i, err)
			}
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/reconciler/sensor/... -v`
Expected: PASS (all tests in the package, including the pre-existing ones)

- [ ] **Step 5: Commit**

```bash
git add pkg/reconciler/sensor/validate.go pkg/reconciler/sensor/validate_test.go
git commit -m "feat(sensors): validate the GRPC trigger CRD spec"
```

---

## Task 6: Wire the trigger into the sensor's trigger registry

**Files:**
- Modify: `pkg/sensors/context.go` (add `grpcTriggerClients` cache)
- Modify: `pkg/sensors/trigger.go` (add the dispatcher case)

**Interfaces:**
- Consumes: `grpctrigger.NewGRPCTrigger` (Task 3/4, package `github.com/argoproj/argo-events/pkg/sensors/triggers/grpc`). `sharedutil.StringKeyedMap[*grpc.ClientConn]` (existing, already imported in `context.go` for `customTriggerClients`).
- Produces: `SensorContext.grpcTriggerClients`, consumed only within `pkg/sensors/trigger.go`'s `GetTrigger`.

- [ ] **Step 1: Add the cache field to `SensorContext`**

In `pkg/sensors/context.go`, add a field after `customTriggerClients` (line 56):

```go
	// customTriggerClients holds the references to the gRPC clients for the custom trigger servers
	customTriggerClients sharedutil.StringKeyedMap[*grpc.ClientConn]
	// grpcTriggerClients holds the references to the gRPC clients for GRPC triggers
	grpcTriggerClients sharedutil.StringKeyedMap[*grpc.ClientConn]
```

And initialize it in `NewSensorContext` (line 86), right after `customTriggerClients`:

```go
		httpClients:          sharedutil.NewStringKeyedMap[*http.Client](),
		customTriggerClients: sharedutil.NewStringKeyedMap[*grpc.ClientConn](),
		grpcTriggerClients:   sharedutil.NewStringKeyedMap[*grpc.ClientConn](),
```

- [ ] **Step 2: Add the dispatcher case**

In `pkg/sensors/trigger.go`, add the import (alphabetically among the other trigger imports, after `"github.com/argoproj/argo-events/pkg/sensors/triggers/email"` and before `"github.com/argoproj/argo-events/pkg/sensors/triggers/http"`):

```go
	grpctrigger "github.com/argoproj/argo-events/pkg/sensors/triggers/grpc"
```

Add the dispatcher case in `GetTrigger`, after the `trigger.Template.CustomTrigger != nil` block (before `if trigger.Template.Log != nil {`):

```go
	if trigger.Template.CustomTrigger != nil {
		result, err := customtrigger.NewCustomTrigger(sensorCtx.sensor, trigger, log, sensorCtx.customTriggerClients)
		if err != nil {
			log.Errorw("failed to new a Custom trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.GRPC != nil {
		result, err := grpctrigger.NewGRPCTrigger(sensorCtx.grpcTriggerClients, sensorCtx.sensor, trigger, log)
		if err != nil {
			log.Errorw("failed to new a GRPC trigger", zap.Error(err))
			return nil
		}
		return result
	}

	if trigger.Template.Log != nil {
```

- [ ] **Step 3: Build and run the sensors package test suite**

Run: `go build ./... && go test ./pkg/sensors/... -v`
Expected: build exits 0; existing sensor tests still pass (this task adds no new test file — `GetTrigger`'s other 10+ branches have no dedicated dispatcher-level test in this codebase either, so this stays consistent with existing convention; correctness of the GRPC branch itself is already covered by Task 3/4's tests exercising `NewGRPCTrigger` directly).

- [ ] **Step 4: Commit**

```bash
git add pkg/sensors/context.go pkg/sensors/trigger.go
git commit -m "feat(sensors): wire the GRPC trigger into the trigger registry"
```

---

## Task 7: Documentation and example

**Files:**
- Create: `docs/sensors/triggers/grpc-trigger.md`
- Modify: `mkdocs.yml:96-109` (add nav entry)
- Create: `examples/sensors/grpc-trigger.yaml`

**Interfaces:**
- None — this task is documentation only, no code interfaces produced or consumed.

- [ ] **Step 1: Write the example Sensor YAML**

Create `examples/sensors/grpc-trigger.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: grpc-trigger
spec:
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
  triggers:
    - template:
        name: grpc-trigger
        grpc:
          url: greeter.argo-events.svc:9000
          method: /helloworld.Greeter/SayHello
          insecure: true
          schema:
            - name: name
              number: 1
              type: string
          payload:
            - src:
                dependencyName: test-dep
                dataKey: body.name
              dest: name
      retryStrategy:
        steps: 3
        duration: 3s
      policy:
        status:
          allow:
            - 0
```

- [ ] **Step 2: Write the doc page**

Create `docs/sensors/triggers/grpc-trigger.md`:

```markdown
# GRPC Trigger

Argo Events offers a GRPC trigger which can invoke an arbitrary unary gRPC method on a target server, without requiring any compiled Go client stubs for that service.

Unlike the [Custom Trigger](build-your-own-trigger.md), which requires the *target* server to implement an Argo-Events-defined proto contract, the GRPC trigger calls services that already exist and know nothing about Argo Events — the same relationship the [HTTP Trigger](http-trigger.md) has to an arbitrary REST endpoint.

## Specification

The GRPC trigger specification is available [here](../../APIs.md#argoproj.io/v1alpha1.GRPCTrigger).

## How it works

Because no compiled stubs exist for the target service, the GRPC trigger needs to be told the shape of the request message. You provide a minimal `schema`: one entry per field, with its `name`, protobuf `number`, and scalar `type`.

```yaml
grpc:
  url: greeter.argo-events.svc:9000
  method: /helloworld.Greeter/SayHello
  insecure: true
  schema:
    - name: name
      number: 1
      type: string
  payload:
    - src:
        dependencyName: test-dep
        dataKey: body.name
      dest: name
```

- `url` is the target gRPC server address (`host:port`).
- `method` is the fully-qualified RPC method path: `/<package>.<Service>/<Method>`.
- `schema` describes the request message fields. Supported `type` values are `string`, `int32`, `int64`, `uint32`, `uint64`, `float`, `double`, `bool`, and `bytes`. Nested messages and repeated fields are not supported.
- `payload` works exactly like every other trigger's `payload`: a list of `src`/`dest` pairs that build the request from event data. Each `dest` should match a `name` in `schema`.
- `insecure: true` connects over plaintext, with no encryption or verification — suitable only for trusted internal networks. Otherwise, set `tls` (same structure as the [HTTP trigger's TLS config](http-trigger.md)) for TLS/mTLS. Exactly one of `tls` or `insecure` must be set.

### ⚠️ Getting the field `number` right

The `number` for each schema field **must exactly match the field number declared in the target service's real `.proto` file** — not the order you happen to list fields in. Protobuf's wire format is is keyed by field number, not by name: if you get a number wrong, the server will not error, it will silently decode your value into the *wrong field* (or drop it as unknown).

To find the correct numbers, either read the target's `.proto` source directly, or if the target has [server reflection](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md) enabled, run:

```bash
grpcurl -plaintext localhost:9000 describe helloworld.Greeter.SayHelloRequest
```

which prints each field's name, type, and number.

## Response and Policy

The GRPC trigger never decodes the response body — only the gRPC status code is used. Use the standard trigger `policy.status.allow` field to determine success, with gRPC status codes (integers, e.g. `0` = `OK`, `5` = `NOT_FOUND`, `14` = `UNAVAILABLE`) instead of HTTP status codes:

```yaml
      policy:
        status:
          allow:
            - 0
```

If `policy` is omitted, no status check is performed and the trigger is always considered successful once the RPC completes (regardless of status code). Combine with `retryStrategy` the same way you would for any other trigger to retry on transient failures.

## Parameterization

Like every other trigger, the GRPC trigger supports [parameterization](https://argoproj.github.io/argo-events/tutorials/02-parameterization/) via its `parameters` field, letting you override `url`, `method`, or any other field from event data at runtime.
```

- [ ] **Step 3: Add the mkdocs navigation entry**

In `mkdocs.yml`, add a line after `"sensors/triggers/http-trigger.md"` (line 100):

```yaml
              - "sensors/triggers/http-trigger.md"
              - "sensors/triggers/grpc-trigger.md"
```

- [ ] **Step 4: Verify the docs build (if `mkdocs` is available)**

Run: `mkdocs build --strict 2>&1 | tail -30`
Expected: exits 0, no broken-link warnings for the new page. If `mkdocs` isn't installed locally, skip this step — it's not required for the Go build/tests to pass, and CI will catch any doc-build regression.

- [ ] **Step 5: Commit**

```bash
git add docs/sensors/triggers/grpc-trigger.md mkdocs.yml examples/sensors/grpc-trigger.yaml
git commit -m "docs(sensors): document the GRPC trigger"
```

---

## Final verification

- [ ] **Step 1: Full build and test suite**

Run: `go build ./... && go test ./... 2>&1 | tail -60`
Expected: build exits 0; no test failures (pre-existing failures unrelated to this change, if any, are out of scope).

- [ ] **Step 2: Lint**

Run: `make lint`
Expected: exits 0, no new lint findings in the files this plan touched.
