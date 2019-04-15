# Slack

The gateway listens to events from Slack.

### How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/30eaa296651e80b11ffef3b20464a08a2041eb09/gateways/community/slack/config.go#L46-L49).

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from Slack.

Create a K8s token that contains your slack `token` and refer that secret in gateway configmap.

**Note:** The gateway will not register the webhook endpoint on Slack. You need to manually do it.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 


## Setup

**1. Install [Gateway](../../examples/gateways/slack.yaml)**

Make sure gateway pod and service is running

**2. Install [Gateway Configmap](../../examples/gateways/slack-gateway-configmap.yaml)**

**3. Install [Sensor](../../examples/sensors/slack.yaml)**

Make sure sensor pod is created.

**4. Trigger Workflow**

A workflow will be triggered when slack sends an event.

