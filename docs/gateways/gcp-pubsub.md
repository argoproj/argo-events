# GCP PubSub

GCP PubSub gateway listens to event streams on google cloud pub sub topics.


### How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/community/gcp-pubsub/config.go#L31-L36).

Make sure to mount credentials file for authentication in gateway pod and refer the path in `credentialsFile`.

## Setup
**1. Install [Event Source](../../examples/event-sources/gcp-pubsub.yaml)**

**2. Install [Gateway](../../examples/gateways/gcp-pubsub.yaml)**

Make sure gateway pod and service is running

**3. Install [Sensor](../../examples/sensors/gcp-pubsub.yaml)**

Make sure sensor pod is created.

**4. Trigger Workflow**

As soon as there a message is consumed from PubSub topic, a workflow will be triggered.
