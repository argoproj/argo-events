# Github

The gateway listens to events from GitHub. 

## Events types and webhook
Refer [here](https://developer.github.com/v3/activity/events/types/) for more information on type of events.

Refer [here](https://developer.github.com/v3/repos/hooks/#get-single-hook) to understand the structure of webhook.

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from GitHub.
The event payload dispatched from gateway contains the type of the event in the headers.

## How to get the URL for the service?
Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup
1. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/github.yaml) before creating the event source configmap, because you need to have the gateway pod running and a service backed by the pod, so that you can get the URL for the service. 

2. Create the [event source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/github.yaml).

3. Deploy the [Sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/github.yaml).

## Trigger Workflow
Depending upon the event you subscribe to, a workflow will be triggered.
