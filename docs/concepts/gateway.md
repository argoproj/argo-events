# Gateway

## What is a gateway?
A gateway consumes events from outside entities, transforms them into the [cloudevents specification](https://github.com/cloudevents/spec) compliant events and dispatches them to sensors.

<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/gateway.png?raw=true" alt="Gateway"/>
</p>

<br/>

There are two components for a gateway,

## Gateway Client
Gateway client manages the event source for the gateway.

Its responsibilities are,

1. Monitor and manage the event sources.
2. Monitor and manage the subscribers.
3. Convert the events received from the gateway server into CloudEvents.
4. Dispatch the cloudevents to subscribers.

## Gateway Server
Gateway server listens to events from event sources.

Its responsibilities are,

1. Validate an event source.
2. Implement the logic for consuming events from an event source.
3. Dispatch events to gateway client.

## Gateway & Event Source
Event Source are event configuration store for a gateway. The configuration stored in an Event Source is used by a gateway to consume events from
external entities like AWS SNS, SQS, GCP PubSub, Webhooks etc.

## Specification
Complete specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/gateway.md).

## Examples
Examples are located under `examples/gateways`.
