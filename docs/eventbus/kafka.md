Kafka is a widely used event streaming platform. We recommend using Kafka if
you have a lot of events and want to horizontally scale your Sensors. If you
are looking to get started quickly with Argo Events we recommend using
Jetstream instead.

When using a Kafka EventBus you must already have a Kafka cluster set up and
topics created (unless you have auto create enabled, see [topics](#topics)
below).

## Example
```yaml
kind: EventBus
metadata:
  name: default
spec:
  kafka:
    url: kafka:9092   # must be managed independently
    topic: "example"  # optional
```

See [here](https://github.com/argoproj/argo-events/blob/master/api/event-bus.md#kafkabus)
for the full specification.

## Properties
### url
Comma seperated list of kafka broker urls, the kafka broker must be managed
independently of Argo Events.

### topic
The topic name, defaults to `{namespace-name}-{eventbus-name}`. Two additional
topics per Sensor are also required, see see [topics](#topics) below for more
information.

### version
Kafka version, we recommend not manually setting this field in most
circumstances. Defaults to the oldest supported stable version.

### tls
Enables TLS on the kafka connection.
```
tls:
  caCertSecret:
    name: my-secret
    key: ca-cert-key
  clientCertSecret:
    name: my-secret
    key: client-cert-key
  clientKeySecret:
    name: my-secret
    key: client-key-key
```

### sasl
Enables SASL authentication on the kafka connection.
```
sasl:
  mechanism: PLAIN
  passwordSecret:
    key: password
    name: my-user
  userSecret:
    key: user
    name: my-user
```

### consumerGroup.groupName
Consumer group name, defaults to `{namespace-name}-{sensor-name}`.

### consumerGroup.rebalanceStrategy
The kafka rebalance strategy, can be one of: sticky, roundrobin, range.
Defaults to range.

### consumerGroup.startOldest
When starting up a new group do we want to start from the oldest event
(true) or the newest event (false). Defaults to false

## Security
You can enable TLS or SASL authentication, see above for configuration
details. You must enable these features in your Kafka Cluster and make
the certifactes/credentials available in a Kubernetes secret.

## Topics
The Kafka EventBus requires one event topic and two additional topics (trigger
and action) per Sensor. These topics will not be created automatically unless
the Kafka `auto.create.topics.enable` cluster configuration is set to true,
otherwise it is your responsibility to create these topics. If a topic does
not exist and cannot be automatically created, the EventSource and/or Sensor
will exit with an error.

If you want to take advantage of the horizontal scaling enabled by the Kafka
EventBus be sure to create topics with more than one partition.

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

Sensors that use a Kafka EventBus can scale horizontally. Specifiying replicas
greater than one will result in all Sensor pods actively processing events.
However, an EventSource that uses a Kafka EventBus cannot necessarily be
horizontally scaled in an active-active manner, see [EventSource HA](../eventsources/ha.md)
for more details. In an active-passive scenario a [Kubernetes leader election](../eventsources/ha.md#kubernetes-leader-election)
is used.
