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

### How Each Topic Is Used

Argo Events uses three Kafka topics per Sensor to implement reliable event
delivery:

- **Event topic** — The shared communication channel between EventSources and
  Sensors. EventSources **produce** CloudEvents messages to this topic, keyed
  by `{source}.{subject}`. Sensors **consume** from it using a consumer group
  to receive the events that match their dependency definitions. All
  EventSources and Sensors that share the same EventBus write to and read from
  the same event topic.

- **Trigger topic** — An internal topic used by the Sensor for **transaction
  coordination**. When a Sensor evaluates its dependency expression and decides
  which triggers to fire, it writes the trigger evaluation state to this topic.
  This enables exactly-once processing semantics across Sensor replicas and
  ensures that triggers are not evaluated more than once for the same set of
  events, even when multiple Sensor pods are running.

- **Action topic** — An internal topic used by the Sensor to **record completed
  actions**. After a trigger action (e.g., creating a Workflow, sending an HTTP
  request) has been executed, the result is written to this topic. This
  provides delivery guarantees and allows the Sensor to track which actions
  have already been performed, preventing duplicate execution on restart or
  rebalance.

The trigger and action topics are specific to each Sensor, so they do not
interfere with other Sensors sharing the same EventBus.

### Topic Naming

By default the topics are named as follows.

| topic   | name                                                | used by          |
| ------- | --------------------------------------------------- | ---------------- |
| event   | `{namespace}-{eventbus-name}`                       | EventSource + Sensor |
| trigger | `{namespace}-{eventbus-name}-{sensor-name}-trigger` | Sensor only      |
| action  | `{namespace}-{eventbus-name}-{sensor-name}-action`  | Sensor only      |

If a topic name is specified in the EventBus specification, then the topics are
named as follows.

| topic   | name                                       | used by          |
| ------- | ------------------------------------------ | ---------------- |
| event   | `{spec.kafka.topic}`                       | EventSource + Sensor |
| trigger | `{spec.kafka.topic}-{sensor-name}-trigger` | Sensor only      |
| action  | `{spec.kafka.topic}-{sensor-name}-action`  | Sensor only      |

### Partitioning Recommendations

If you want to take advantage of the horizontal scaling enabled by the Kafka
EventBus, create topics with more than one partition. Here are some guidelines:

- **Event topic** — The number of partitions determines the maximum parallelism
  for Sensor consumers. Set the partition count to at least the number of
  Sensor replicas you plan to run. If you expect high event throughput, use
  more partitions to distribute the load.

- **Trigger and action topics** — These are used for internal coordination and
  typically have lower throughput than the event topic. A small number of
  partitions (e.g., 1-3) is generally sufficient.

### Single vs. Multiple EventBus

A single EventBus (and therefore a single event topic) is sufficient for most
use cases, even when you have EventSources producing different types of events.
Events are keyed by `{source}.{subject}`, and each Sensor filters only the
events matching its dependency definitions.

Consider using **separate EventBus** resources (and therefore separate event
topics) when:

- You want to isolate event traffic between teams or environments.
- Different event streams have significantly different throughput requirements
  or retention policies.
- You need different security configurations (TLS, SASL) for different event
  producers.

## Horizontal Scaling and Leader Election

Sensors that use a Kafka EventBus can scale horizontally. Specifying replicas
greater than one will result in all Sensor pods actively processing events.
However, an EventSource that uses a Kafka EventBus cannot necessarily be
horizontally scaled in an active-active manner, see [EventSource HA](../eventsources/ha.md)
for more details. In an active-passive scenario a [Kubernetes leader election](../eventsources/ha.md#kubernetes-leader-election)
is used.
