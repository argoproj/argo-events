# Streams

A Stream Gateway basically listens to messages on a message queue.

The configuration for an event source is somewhat similar between all stream gateways. We will go through setup of NATS gateway.

## NATS

NATS gateway consumes messages by subscribing to NATS topic.

## Setup

1. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/nats.yaml).

2. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/nats.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/nats.yaml).

## Trigger Workflow
Publish message to subject `foo`. You might find [this](https://github.com/nats-io/go-nats/tree/master/examples/nats-pub) useful.
