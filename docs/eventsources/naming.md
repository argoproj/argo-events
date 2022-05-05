# EventSource Names

In a Sensor object, a `dependency` is defined as:

```yaml
dependencies:
  - name: test-dep
    eventSourceName: webhook-example
    eventName: example
```

The `eventSourceName` and `eventName` might be confusing. Take the following
EventSource example, the `eventSourceName` and `eventName` are described as
below.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook-example # eventSourceName
spec:
  webhook:
    example: # eventName
      port: "12000"
      endpoint: /example
      method: POST
    example-foo: # eventName
      port: "13000"
      endpoint: /example2
      method: POST
```

## EventSourceName

`eventSourceName` is the `name` of the dependent `EventSource` object, i.e.
`webhook-example` in the example above.

## EventName

`eventName` is the map key of a configured event. In the example above,
`eventName` could be `example` or `example-foo`.
