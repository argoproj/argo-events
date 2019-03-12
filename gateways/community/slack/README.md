<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/slack.png?raw=true" alt="Slack"/>
</p>

<br/>

# Slack

The gateway listens to events from Slack.

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Token for URL verification handshake
Token *corev1.SecretKeySelector `json:"token"`

// Webhook
Hook *common.Webhook `json:"hook"`
```

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: slack-gateway-configmap
data:
  notification_1: |-
    hook:
     endpoint: "/event"
     port: "12000"
     url: "URL_OF_SERVICE"
    token:
      name slack-secret
      key: tokenkey
```

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from Slack.

Create a K8s token that contains your slack `token` and refer that secret in gateway configmap.

**Note:** The gateway will not register the webhook endpoint on Slack. You need to manually do it.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 


## Setup

**1. Install Gateway**
```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/slack.yaml
```

Make sure gateway pod and service is running

**2. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/slack-gateway-configmap.yaml
```

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/slack.yaml
```

Make sure sensor pod is created.

**4. Trigger Workflow**

As soon as there a message is consumed from SQS queue, a workflow will be triggered.

