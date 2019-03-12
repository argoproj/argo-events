<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/pubsub.png?raw=true" alt="GCP PubSub"/>
</p>

<br/>

# GCP PubSub

GCP PubSub gateway listens to event streams on google cloud pub sub topics.


### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
	// ProjectID is the unique identifier for your project on GCP
	ProjectID string `json:"projectID"`
	// Topic on which a subscription will be created
	Topic string `json:"topic"`
	// CredentialsFile is the file that contains credentials to authenticate for GCP
	CredentialsFile string `json:"credentialsFile"`
```

Make sure to mount credentials file for authentication in gateway pod and refer the path in `credentialsFile`.

### Example 

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gcp-pubsub-gateway-configmap
data:
  notification_1: |-
    projectId: "your-project-id"
    topic: "your-topic"
    credentialsFile: "path to credential files"
```

## Setup
**1. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/gcp-pubsub-gateway-configmap.yaml
```

**2. Install Gateway**
```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/gcp-pubsub.yaml
```

Make sure gateway pod and service is running

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/gcp-pubsub.yaml
```

Make sure sensor pod is created.

**4. Trigger Workflow**

As soon as there a message is consumed from PubSub topic, a workflow will be triggered.
