<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/trello.png?raw=true" alt="GitHub"/>
</p>

<br/>


# Trello

The gateway listens to events from Trello. 

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Webhook
Hook *common.Webhook `json:"hook"`

// ApiKey for client
ApiKey *corev1.SecretKeySelector `json:"apiKey"`

// Token for client
Token *corev1.SecretKeySelector `json:"token"`

// Description for webhook
// +optional
Description string `json:"description,omitempty"`
```

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: trello-gateway-configmap
data:
  notification_1: |-
    hook:
     endpoint: "/test"
     port: "12000"
     url: "placeholderurl"
    apiKey:
      name: trello-secret
      key: apikey
    token:
      name trello-secret
      key: tokenkey
    description: "test webhook"
```

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from Trello.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

**1. Install Gateway**

We are installing gateway before creating configmap. Because we need to have the gateway pod running and a service backed by the pod, so 
that we can get the URL for the service. 

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/trello.yaml
```

Make sure gateway pod and service is running

**2. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/trello-gateway-configmap.yaml
```

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/trello.yaml
```

Make sure sensor pod is created.

**4. Trigger Workflow**

Depending upon the event you subscribe to, a workflow will be triggered.
