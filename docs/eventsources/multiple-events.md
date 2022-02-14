# EventSource With Multiple Events

![GA](../assets/ga.svg)

> v0.17.0 and after

Multiple events can be configured in a single EventSource, they can be either
one event source type, or mixed event source types with some limitations.

## Single EventSource Type

A single type EventSource configuration:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: webhook
spec:
  webhook:
    example:
      port: "12000"
      endpoint: /example
      method: POST
    example-foo:
      port: "13000"
      endpoint: /example2
      method: POST
```

For the example above, there are 2 events configured in the EventSource named
`webhook`.

## Mixed EventSource Types

EventSource is allowed to have mixed types of events configured.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: mixed-sources
spec:
  webhook:
    webhook-example: # eventName
      port: "12000"
      endpoint: /example
      method: POST
  sns:
    sns-example: # eventName
      topicArn: arn:aws:sns:us-east-1:XXXXXXXX:test
      webhook:
        endpoint: "/"
        port: "15000"
      accessKey:
        key: my-key
        name: my-name
      secretKey:
        key: my-secret-key
        name: my-secret-name
      region: us-east-1
```

However, there are some rules need to follow to do it:

- EventSource types with `Active-Active` HA strategy can not be mixed with types
  with `Active-Passive` strategy, for EventSource types, see
  [EventSource High Availability](ha.md) for the detail.

- Event Name (i.e. `webhook-example` and `sns-example` above, refer to
  [EventSource Names](naming.md)) needs to be unique in the EventSource, same
  `eventName` is not allowed even they are in different event source types.

  The reason for that is, we use `eventSourceName` and `eventName` as the
  dependency attributes in Sensor.
