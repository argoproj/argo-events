# Gateway

## What is a gateway?
A gateway consumes events from outside entities, transforms them into the [cloudevents specification](https://github.com/cloudevents/spec) compliant events and dispatches them to sensors.

<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/gateway.png?raw=true" alt="Gateway"/>
</p>

<br/>

## Gateway & Event Source relation
Event Source are configuration store for a gateway to select from. The configuration stored in an Event Source is used by a gateway to consume events from
external entities like AWS SNS, SQS, GCP PubSub, Webhooks etc.

## Specification
Complete specification is available [here]()

