# EventSource With Multiple Events

![alpha](assets/alpha.svg)

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
`webhook`. Please use different `port` numbers for different events, this is the
limitation for multiple events configured in a `webhook` EventSource, this
limitation also applies to `webhook` extended event source types such as
`github`, `sns`.

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

- `Rolling Update` types and `Recreate` types can not be configured together,
  see [EventSource Deployment Strategies](eventsource-deployment-strategies.md).

- Event Name (i.e. `webhook-example` and `sns-example` above, refer to
  [EventSource Names](eventsource-names.md)) needs to be unique in the
  EventSource, same `eventName` is not allowed even they are in different event
  source types.

  The reason for that is, we use `eventSourceName` and `eventName` as the
  dependency attributes in Sensor.
