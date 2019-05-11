# Gitlab

The gateway listens to events from Gitlab. 

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from Gitlab.

## How to get the URL for the service?
Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

1. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/gitlab.yaml) before creating the event source configmap, 
because you need to have the gateway pod running and a service backed by the pod, so that you can get the URL for the service. 

2. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/gitlab.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/gitlab.yaml).

## Trigger Workflow.
Depending upon the event you subscribe to, a workflow will be triggered.

