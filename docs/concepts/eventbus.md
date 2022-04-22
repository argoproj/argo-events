# Eventbus

The eventbus acts as the transport layer of Argo-Events by connecting the event-sources and sensors.

Event-Sources publish the events while the sensors subscribe to the events to execute triggers.

There are two implementations of the eventbus: [NATS streaming](https://docs.nats.io/legacy/stan/intro#:~:text=NATS%20Streaming%20is%20a%20data,with%20the%20core%20NATS%20platform.) and now [NATS Jetstream](https://docs.nats.io/nats-concepts/jetstream) (which will replace the former, which will be deprecated).
