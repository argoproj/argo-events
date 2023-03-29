Kafka is a widely used event streaming platform. Unlike NATs and Jetstream, a
Kafka cluster managed independently of Argo Events is required to use a Kafka
EventBus.

## Example
```yaml
kind: EventBus
metadata:
  name: default
spec:
  kafka:
    url: kafka:9092   # must be deployed independently
    topic: "example"  # optional
```

See [here](https://github.com/argoproj/argo-events/blob/master/api/event-bus.md#kafkabus)
for the full specification.

## Topics

The Kafka EventBus requires one event topic and two additional topics (trigger
and action) per Sensor. These topics will not be created automatically unless
the Kafka `auto.create.topics.enable` cluster configuration is set to true,
otherwise it is your responsibility to create these topics. If a topic does
not exist and cannot be automatically created, the EventSource and/or Sensor
will exit with an error.

By default the topics are named as follows.

| topic | name |
| ----- | ---- |
| event | `{namespace}-{eventbus-name}` |
| trigger | `{namespace}-{eventbus-name}-{sensor-name}-trigger` |
| action | `{namespace}-{eventbus-name}-{sensor-name}-action` |

If a topic name is specified in the EventBus specification, then the topics are
named as follows.

| topic | name |
| ----- | ---- |
| event | `{spec.kafka.topic}` |
| trigger | `{spec.kafka.topic}-{sensor-name}-trigger` |
| action | `{spec.kafka.topic}-{sensor-name}-action` |

## Horizontal Scaling and Leader Election

Sensors that use a Kafka EventBus can scale horizontally. Unlike NATs and
Jetstream, specifiying a replicas value greater than 1 will result in all
Sensor pods actively processing events. However, an EventSource that uses a
Kafka EventBus cannot be horizontally scaled and a
[kubernetes leader election](https://argoproj.github.io/argo-events/eventsources/ha/#kubernetes-leader-election) is
used.
