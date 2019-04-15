# Gitlab

The gateway listens to events from Gitlab. 

Refer [here](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html#events) for more information on type of events.

### How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/30eaa296651e80b11ffef3b20464a08a2041eb09/gateways/community/gitlab/config.go#L49-L63).

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from GitHub.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

**1. Install [Gateway](../../../examples/gateways/gitlab.yaml)**

We are installing gateway before creating configmap. Because we need to have the gateway pod running and a service backed by the pod, so 
that we can get the URL for the service. 

Make sure gateway pod and service is running

**2. Install [Gateway Configmap](../../../examples/gateways/gitlab-gateway-configmap.yaml)**

**3. Install [Sensor](../../../examples/sensors/gitlab.yaml)**

Make sure sensor pod is created.

**4. Trigger Workflow**

Depending upon the event you subscribe to, a workflow will be triggered.

