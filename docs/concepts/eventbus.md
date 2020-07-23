# Eventbus

The eventbus acts as the transport layer of Argo-Events by connecting the event-sources and sensors.

Event-Sources publish the events while the sensors subscribe to the events to execute triggers.

The current implementation of the eventbus is powered by NATS streaming.
