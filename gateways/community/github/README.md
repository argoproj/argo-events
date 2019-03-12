<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/github.png?raw=true" alt="GitHub"/>
</p>

<br/>

# Github

The gateway listens to events from GitHub. 

Refer [here](https://developer.github.com/v3/activity/events/types/) for more information on type of events.

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Webhook
Hook *common.Webhook `json:"hook"`

// GitHub owner name i.e. argoproj
Owner string `json:"owner"`

// GitHub repo name i.e. argo-events
Repository string `json:"repository"`

// Github events to subscribe to which the gateway will subscribe
Events []string `json:"events"`

// K8s secret containing github api token
APIToken *corev1.SecretKeySelector `json:"apiToken"`

// K8s secret containing WebHook Secret
WebHookSecret *corev1.SecretKeySelector `json:"webHookSecret"`

// Insecure tls verification
Insecure bool `json:"insecure"`

// Active
Active bool `json:"active"`

// ContentType json or form
ContentType string `json:"contentType"`
```

Refer [this](https://developer.github.com/v3/repos/hooks/#get-single-hook) to understand the structure of webhook.

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: github-gateway-configmap
data:
  project_1: |-
    owner: "owner-example"
    repository: "repo-example"
    hook:
     endpoint: "/push"
     port: "12000"
     url: "YOUR_SERVICE_URL"
    events:
    - "*"
    apiToken:
      name: github-access
      key: token
    webHookSecret:
      name: github-access
      key: secret
    insecure: false
    active: true
    contentType: "json"
```

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from GitHub.

**How to get the URL for the service?**
Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

**1. Install Gateway**

We are installing gateway before creating configmap. Because we need to have the gateway pod running and a service backed by the pod, so 
that we can get the URL for the service. 

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/github.yaml
```

Make sure gateway pod and service is running

**2. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/github-gateway-configmap.yaml
```

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/github.yaml
```

Make sure sensor pod is created.

**4. Trigger Workflow**

Depending upon the event you subscribe to, a workflow will be triggered.
