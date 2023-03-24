# EventBus

The EventBus acts as the transport layer of Argo-Events by connecting the EventSources and Sensors.

EventSources publish the events while the Sensors subscribe to the events to execute triggers.

There are three implementations of the EventBus: [NATS](https://docs.nats.io/legacy/stan/intro#:~:text=NATS%20Streaming%20is%20a%20data,with%20the%20core%20NATS%20platform.) (deprecated), [Jetstream](https://docs.nats.io/nats-concepts/jetstream), and [Kafka](https://kafka.apache.org).
