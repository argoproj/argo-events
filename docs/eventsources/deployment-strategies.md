# EventSource Deployment Strategies

EventSource controller creates a k8s deployment for each EventSource object to
watch the events. Some of the event source types do not allow multiple live
clients with same attributes (i.e. multiple clients with same `clientID`
connecting to a NATS server), or multiple event source PODs will generate
duplicated events to downstream, so the deployment strategy and replica numbers
are different for different event sources.

## Rolling Update Strategy

`Rolling Update` strategy is applied to the following EventSource types:

- AWS SNS
- AWS SQS
- Github
- Gitlab
- NetApp Storage GRID
- Slack
- Stripe
- Webhook

### Replicas Of Rolling Update Types

Deployment replica of these event source types respects `spec.replica` in the
EventSource object, defaults to 1.

## Recreate Strategy

`Recreate` strategy is applied to the following EventSource types:

- AMQP
- Azure Events Hub
- Kafka
- GCP PubSub
- File
- HDFS
- NATS
- Minio
- MQTT
- Emitter
- NSQ
- Pulsar
- Redis
- Resource
- Calendar
- Generic

### Replicas Of Recreate Types

`spec.replica` in the `Recreate` types EventSources is ignored, the deployment
is always created with `replica=1`.

**Please DO NOT manually scale up the replicas, that will lead to unexpected
behaviors!**
