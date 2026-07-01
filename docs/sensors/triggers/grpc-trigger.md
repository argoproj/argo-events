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
