# GCP PubSub

The gateway listens to event streams from google cloud pub sub topics.

Make sure to mount credentials file for authentication in gateway pod and refer the path in `credentialsFile`.

## Setup
1. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/gcp-pubsub.yaml).

2. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/gcp-pubsub.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/gcp-pubsub.yaml).

## Trigger Workflow
As soon as there a message is consumed from PubSub topic, a workflow will be triggered.
