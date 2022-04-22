# Eventbus

The eventbus acts as the transport layer of Argo-Events by connecting the event-sources and sensors.

Event-Sources publish the events while the sensors subscribe to the events to execute triggers.

There are two implementations of the eventbus: NATS streaming and now NATS Jetstream.
