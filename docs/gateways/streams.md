# Streams

A Stream Gateway basically listens to messages on a message queue.

The configuration for an event source is somewhat similar between all stream gateways. We will go through setup of NATS gateway.

## NATS

NATS gateway consumes messages by subscribing to NATS topic.

### How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/39fd5b8592e27c869dcc6cf2b0e6fee1d56622f2/gateways/core/stream/nats/config.go#L33-L40).

If you don't provide an explicit `Backoff`, then gateway will use a default backoff in case of connection failure.

The [event source](../../examples/gateways/nats-gateway-configmap.yaml) contains information to subscribe to topic `foo`, it also contains an explicit backoff option
in case of the connection failure.

### Setup

**1. Install [Gateway Configmap](../../examples/gateways/nats-gateway-configmap.yaml)**

**2. Install [Gateway](../../examples/gateways/nats.yaml)**

Make sure the gateway pod is created.

**3. Install [Sensor](../../examples/sensors/nats.yaml)**

Make sure the sensor pod is created

**4. Trigger Workflow**

Publish message to subject `foo`. 
You might find this useful https://github.com/nats-io/go-nats/tree/master/examples/nats-pub
