# Github

The gateway listens to events from GitHub. 

Refer [here](https://developer.github.com/v3/activity/events/types/) for more information on type of events.

### How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/30eaa296651e80b11ffef3b20464a08a2041eb09/gateways/community/github/config.go#L50-L73).

Refer [this](https://developer.github.com/v3/repos/hooks/#get-single-hook) to understand the structure of webhook.

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from GitHub.

The event payload dispatched from gateway contains the type of the event in the headers.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

**1. Install [Gateway](../../examples/gateways/github.yaml)**

We are installing gateway before creating configmap. Because we need to have the gateway pod running and a service backed by the pod, so 
that we can get the URL for the service. 

Make sure gateway pod and service is running

**2. Install [Gateway Configmap](../../examples/event-sources/github.yaml)**

**3. Install [Sensor](../../examples/sensors/github.yaml)**

Make sure sensor pod is created.

**4. Trigger Workflow**

Depending upon the event you subscribe to, a workflow will be triggered.
