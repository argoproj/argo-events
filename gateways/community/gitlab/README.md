<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/gitlab.png?raw=true" alt="Gitlab"/>
</p>

<br/>


# Gitlab

The gateway listens to events from Gitlab. 

Refer [here](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html#events) for more information on type of events.

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Webhook
Hook *common.Webhook `json:"hook"`

// ProjectId is the id of project for which integration needs to setup
ProjectId string `json:"projectId"`

// Event is a gitlab event to listen to.
// Refer https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794.
Event string `json:"event"`

// AccessToken is reference to k8 secret which holds the gitlab api access information
AccessToken *GitlabSecret `json:"accessToken"`

// EnableSSLVerification to enable ssl verification
EnableSSLVerification bool `json:"enableSSLVerification"`

// GitlabBaseURL is the base URL for API requests to a custom endpoint
GitlabBaseURL string `json:"gitlabBaseUrl"`
```

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gitlab-gateway-configmap
data:
  project_1: |-
    projectId: "1"
    hook:
     endpoint: "/push"
     port: "12000"
     url: "URL_FOR_SERVICE"
    event: "PushEvents"
    accessToken:
      key: accesskey
      name: gitlab-access
    enableSSLVerification: false
    gitlabBaseUrl: "YOUR_GITLAB_URL"
```

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server and make it reachable from GitHub.

**How to get the URL for the service?**

Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

**1. Install Gateway**

We are installing gateway before creating configmap. Because we need to have the gateway pod running and a service backed by the pod, so 
that we can get the URL for the service. 

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/gitlab.yaml
```

Make sure gateway pod and service is running

**2. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/gitlab-gateway-configmap.yaml
```

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/gitlab.yaml
```

Make sure sensor pod is created.

**4. Trigger Workflow**

Depending upon the event you subscribe to, a workflow will be triggered.

