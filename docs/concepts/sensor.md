# Sensor

Sensor defines a set of event dependencies (inputs) and triggers (outputs).
It listens to events on the eventbus and acts as an event dependency manager to resolve and execute the triggers.

## Event dependency

A dependency is an event the sensor is waiting to happen.

### JetStream Configuration

When using JetStream as the EventBus, you can configure the deliver policy for each dependency using the `jetStream` field. This allows you to control how messages are delivered:

- **`all`**: Default policy. Start receiving from the earliest available message in the stream
- **`last`**: Start with the last message added to the stream, or the last message matching the consumer's filter subject if defined
- **`new`**: Start receiving messages created after the consumer was created

```yaml
spec:
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
      jetStream:
        deliverPolicy: "all" # Start from earliest available message (default)
```

## Specification

Complete specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md).

## Examples

Examples are located under [examples/sensors](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
