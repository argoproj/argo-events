# gRPC Trigger — Design

## Overview

Add a new `GRPC` trigger type to argo-events, structurally analogous to the
existing `HTTP` trigger: a Sensor can invoke an arbitrary **unary** gRPC
method on a target server that was not written with argo-events in mind, no
compiled Go client stubs required.

This is distinct from the existing `custom-trigger` (`pkg/sensors/triggers/custom-trigger/`),
which also uses gRPC but requires the *target* server to implement a fixed
argo-events-defined proto contract (`FetchResource`/`Execute`/`ApplyPolicy`).
The new `GRPC` trigger calls services that already exist and know nothing
about argo-events — the same relationship the `HTTP` trigger has to an
arbitrary REST endpoint.

## v1 Scope

In scope:
- Unary RPCs only (no client/server/bidi streaming)
- Request message built from a **user-supplied minimal schema**
  (`name` + `type` + `number` per field) plus the existing `Payload`
  parameter-mapping mechanism shared with other triggers
- Response is **not decoded** — only the gRPC status code is used, to drive
  the retry/backoff policy
- TLS/mTLS and plaintext (insecure) connections
- Connection reuse via a cached `grpc.ClientConn` pool, keyed by target +
  TLS config, mirroring the existing `customTriggerClients` cache pattern

Explicitly out of scope for v1 (candidates for a future version):
- Server reflection (auto-discovering the schema instead of the user
  supplying it) — an escape hatch (the manual schema) always exists, so
  reflection is a pure UX improvement, not a blocker
- gRPC request metadata / header support (no bearer-token-style auth;
  TLS/mTLS or plaintext-for-internal-use are the only auth paths in v1)
- Nested message fields and repeated fields in the schema (flat scalars only)
- Decoding/exposing the response body

## CRD Schema

New types in `pkg/apis/events/v1alpha1/sensor_types.go`, added alongside the
existing per-trigger-type structs (see `HTTPTrigger`, lines ~399-436, as the
closest existing pattern):

```go
type GRPCTrigger struct {
    // URL is the target gRPC server address (host:port).
    URL string `json:"url"`

    // Method is the fully-qualified RPC method path,
    // e.g. "/helloworld.Greeter/SayHello".
    Method string `json:"method"`

    // Schema describes the request message's fields. Required because no
    // compiled stubs exist for the target service — number MUST match the
    // field number in the target's real .proto definition, or the message
    // will be silently misinterpreted on the wire (protobuf does not
    // validate field numbers against the sender's intent).
    Schema []GRPCSchemaField `json:"schema,omitempty"`

    // Payload is the list of key-value extraction rules from events used to
    // construct the request body. Same mechanism/type as HTTPTrigger.Payload.
    Payload []TriggerParameter `json:"payload"`

    // TLS configuration for the gRPC connection (same type HTTPTrigger uses).
    // Mutually exclusive with Insecure.
    TLS *TLSConfig `json:"tls,omitempty"`

    // Insecure disables TLS entirely (plaintext connection).
    Insecure bool `json:"insecure,omitempty"`

    // Timeout for the RPC call, in seconds. Defaults to 10.
    Timeout int64 `json:"timeout,omitempty"`

    // Parameters is the standard trigger-level parameterization (can
    // override URL/Method/etc from event data), same as other triggers.
    Parameters []TriggerParameter `json:"parameters,omitempty"`
}

type GRPCSchemaField struct {
    // Name is the JSON/proto field name.
    Name string `json:"name"`
    // Number is the protobuf field number, taken from the target's .proto.
    Number int32 `json:"number"`
    // Type is the scalar field type: string, int32, int64, uint32, uint64,
    // float, double, bool, or bytes.
    Type string `json:"type"`
}
```

No new policy/backoff type is introduced. Retry/backoff and success-code checking are
**not** per-trigger-type in this codebase — every trigger type reuses the generic,
already-existing `v1alpha1.Trigger.Policy.Status.Allow []int32` field (checked via
`pkg/sensors/policy.StatusPolicy`, see `pkg/sensors/triggers/http/http.go:214-226`)
plus `v1alpha1.Trigger.RetryStrategy` (a `*Backoff`, applied generically around the
whole Execute+ApplyPolicy attempt in `pkg/sensors/listener.go:215-232`). The GRPC
trigger plugs into that same shared mechanism: `Allow` is interpreted as a list of
numeric gRPC status codes (`codes.Code` is a `uint32` under the hood, e.g. `0` = OK,
`14` = UNAVAILABLE) instead of HTTP status codes. If `Policy` is unset, no check is
performed (same default behavior as every other trigger type).

Also:
- Add `GRPC *GRPCTrigger` field to `TriggerTemplate`.
- Add `TriggerTypeGRPC TriggerType = "GRPC"` constant in `const.go`.

## Request-Building & Invocation Flow

New package `pkg/sensors/triggers/grpc/grpc.go`, implementing the standard
`Trigger` interface (`GetTriggerType`, `FetchResource`,
`ApplyResourceParameters`, `Execute`, `ApplyPolicy`) — same shape as
`pkg/sensors/triggers/http/http.go`.

Inside `Execute`:

1. **Build the message descriptor** from `Schema`: construct a
   `descriptorpb.DescriptorProto` with one `FieldDescriptorProto` per schema
   entry (`Name`, `Number`, and a scalar `Type` mapped from the string type
   name), wrap it in a synthetic `FileDescriptorProto`, and resolve it via
   `protodesc.NewFile` into a `protoreflect.MessageDescriptor`.
2. **Build the request JSON** using the existing `Payload` →
   `TriggerParameter` resolution mechanism — identical to how the HTTP
   trigger builds its JSON body from event data.
3. **Construct the dynamic message**: `dynamicpb.NewMessage(descriptor)`,
   then `protojson.Unmarshal(payloadJSON, dynMsg)`. Uses only the already-
   vendored `google.golang.org/protobuf` stdlib APIs — no new dependency.
4. **Invoke** via `grpc.ClientConn.Invoke(ctx, Method, dynMsg, reply, ...)`.
   Since the response is discarded, the exact placeholder mechanism for the
   reply argument (e.g. a permissive empty descriptor, or a raw-bytes sink
   codec) is an implementation detail to resolve during coding, not a
   design constraint — functionally the call either succeeds or fails with
   a status code.
5. **Capture the returned `status.Status`** (code + message) for `ApplyPolicy`.

**Connection management:** `grpc.ClientConn`s are cached and reused, keyed by
`trigger.Template.Name` (the Sensor trigger's own name) — the exact same
keying scheme `httpClients` and `customTriggerClients` already use (see
`pkg/sensors/triggers/custom-trigger/custom-trigger.go:59,122`). Added to
`pkg/sensors/context.go` as a new `grpcTriggerClients` cache.

**`ApplyPolicy`:** if `Trigger.Policy == nil || Trigger.Policy.Status == nil
|| Trigger.Policy.Status.Allow == nil`, no check is performed (matches every
other trigger type's default). Otherwise, compares the numeric gRPC status
code from the captured `status.Status` against `Trigger.Policy.Status.Allow`
using the existing shared `pkg/sensors/policy.NewStatusPolicy(...).ApplyPolicy(ctx)`
— identical call shape to `HTTPTrigger.ApplyPolicy`, just passing the gRPC
status code integer instead of the HTTP status code. Retry/backoff of the
whole attempt is not this trigger's concern — it comes for free from the
existing generic `Trigger.RetryStrategy` handling in `pkg/sensors/listener.go`.

## Wiring

- `pkg/sensors/trigger.go`: add a case in `GetTrigger()`'s dispatcher for
  `TriggerTypeGRPC`, instantiating the new `grpc.GRPCTrigger` — same pattern
  as the existing `HTTP`/`Kafka`/`CustomTrigger` cases.
- `pkg/sensors/context.go`: add the `grpcTriggerClients` cache described above.
- CRD validation: add a validate function (mirroring `ValidateHTTPTrigger`)
  checking `URL`/`Method` are non-empty, `Schema` field `Type`s are one of
  the supported scalars, `Number`s are positive and unique, and
  `TLS`/`Insecure` aren't both set contradictorily.
- Regenerate CRD manifests / deepcopy (codegen) since this adds new API types.

## Testing

Unit tests in `pkg/sensors/triggers/grpc/grpc_test.go`: spin up an in-process
`grpc.Server` (via `bufconn` or a loopback listener) registered with a small
test service, exercise `FetchResource`/`ApplyResourceParameters`/`Execute`/
`ApplyPolicy` against it — same style as `http_test.go`'s use of
`httptest.Server`. Cover:
- A successful call
- Schema→message round-trip correctness (right field numbers/types reach
  the server correctly)
- TLS and insecure connection paths
- Policy-driven retry on a non-OK status
- A malformed-schema validation-error case

## Documentation

- `docs/sensors/triggers/grpc-trigger.md`, following the existing per-trigger
  doc pattern (`http-trigger.md` as the template): full field reference, a
  worked example, and a **prominent, explicit warning** about how to find
  the correct field `number` for a target proto (e.g. from the `.proto`
  source, or via `grpcurl -describe`) — since a wrong number does not error,
  it silently sends the wrong data on the wire.
- Add an example Sensor YAML under `examples/triggers/`, mirroring the
  existing HTTP trigger example.
