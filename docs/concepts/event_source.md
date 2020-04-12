# Event Source

Event Sources are the configuration store for gateways. The configuration stored in an Event Source is used by a gateway to consume events from
external entities like AWS SNS, SQS, GCP PubSub, Webhooks etc.


Currently supported event sources -

1. AMQP
1. AWS SNS
1. AWS SQS
1. Cron Schedules
1. GCP PubSub
1. GitHub
1. GitLab
1. HDFS
1. File Based Events
1. Kafka
1. Minio
1. NATS
1. MQTT
1. K8s Resources
1. Slack
1. NetApp StorageGrid
1. Webhooks
1. Stripe
1. NSQ
1. Emitter
1. Redis
1. Azure Events Hub


## Specification
Complete specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/event-source.md).

## Examples
Examples are located under `examples/event-sources`.
