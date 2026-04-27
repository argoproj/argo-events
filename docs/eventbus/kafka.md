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
    url: kafka:9092 # must be managed independently
    topic: "example" # optional
```

See [here](../APIs.md#argoproj.io/v1alpha1.KafkaBus)
for the full specification.

## Properties

### url

Comma separated list of kafka broker urls, the kafka broker must be managed
independently of Argo Events.

### topic

The topic name, defaults to `{namespace-name}-{eventbus-name}`. Two additional
topics per Sensor are also required, see [topics](#topics) below for more
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

### partitioner

Producer partitioning strategy. Supported values: `random`, `hash`, `roundrobin`, `manual`. Defaults to `random`.

### consumerBatchMaxWait

Sets the maximum time the sensor consumer will wait to fill a batch of
messages before processing them. Accepts a Go duration string (e.g.,
`1s`, `500ms`, `100ms`). Set to `0` to disable batching and process
messages individually in real-time. Defaults to `1s`.

```yaml
spec:
  kafka:
    url: kafka:9092
    consumerBatchMaxWait: "100ms"  # or "0" for real-time
```

**Performance considerations:**

- Higher values improve throughput by amortizing Kafka transaction
  overhead across more messages, but increase latency.
- Lower values reduce latency at the cost of more frequent transactions.
- Setting to `0` provides the lowest latency (real-time processing).
  Single-dependency triggers bypass the action topic entirely in this
  mode, invoking actions directly without Kafka transactions. Multi-dependency
  triggers still use the trigger topic for dependency aggregation but skip
  the action topic once all dependencies are satisfied.

This value can be overridden per Sensor using the
`eventBusConsumerBatchMaxWait` field in the Sensor spec:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: my-sensor
spec:
  eventBusConsumerBatchMaxWait: "0"  # real-time for this sensor only
```

If both are set, the Sensor-level value takes precedence over the
EventBus-level value.

### transactionRetryBackoff

Sets the backoff duration between Kafka transaction retry attempts when
`CONCURRENT_TRANSACTIONS` errors occur. The Kafka broker returns this error
when a new transaction starts before the previous one is fully finalized.
Lower values reduce latency at the cost of slightly higher CPU from more
frequent retries. Accepts a Go duration string. Defaults to `10ms`.

```yaml
spec:
  kafka:
    url: kafka:9092
    transactionRetryBackoff: "10ms"  # default, optimized for single-region
```

For cross-region deployments where the transaction coordinator has higher
latency, use a larger value:

```yaml
spec:
  kafka:
    url: kafka:9092
    transactionRetryBackoff: "100ms"  # cross-region deployments
```

## Security

You can enable TLS or SASL authentication, see above for configuration
details. You must enable these features in your Kafka Cluster and make
the certificates/credentials available in a Kubernetes secret.

## Topics

The Kafka EventBus requires one event topic and two additional topics (trigger
and action) per Sensor. These topics will not be created automatically unless
the Kafka `auto.create.topics.enable` cluster configuration is set to true,
otherwise it is your responsibility to create these topics. If a topic does
not exist and cannot be automatically created, the EventSource and/or Sensor
will exit with an error.

> **Note:** While all three topics are always created, the action topic is
> bypassed in common flows. Single-dependency triggers invoke actions directly
> without producing to the action topic. Multi-dependency triggers also skip
> the action topic once all dependencies are satisfied. The action topic
> remains as a fallback and is still consumed by the Sensor.

If you want to take advantage of the horizontal scaling enabled by the Kafka
EventBus be sure to create topics with more than one partition.

By default the topics are named as follows.

| topic   | name                                                |
| ------- | --------------------------------------------------- |
| event   | `{namespace}-{eventbus-name}`                       |
| trigger | `{namespace}-{eventbus-name}-{sensor-name}-trigger` |
| action  | `{namespace}-{eventbus-name}-{sensor-name}-action`  |

If a topic name is specified in the EventBus specification, then the topics are
named as follows.

| topic   | name                                       |
| ------- | ------------------------------------------ |
| event   | `{spec.kafka.topic}`                       |
| trigger | `{spec.kafka.topic}-{sensor-name}-trigger` |
| action  | `{spec.kafka.topic}-{sensor-name}-action`  |

## Horizontal Scaling and Leader Election

Sensors that use a Kafka EventBus can scale horizontally. Specifying replicas
greater than one will result in all Sensor pods actively processing events.
However, an EventSource that uses a Kafka EventBus cannot necessarily be
horizontally scaled in an active-active manner, see [EventSource HA](../eventsources/ha.md)
for more details. In an active-passive scenario a [Kubernetes leader election](../eventsources/ha.md#kubernetes-leader-election)
is used.
