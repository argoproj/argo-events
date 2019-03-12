<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/sqs.png?raw=true" alt="AWS SQS"/>
</p>

<br/>

# AWS SQS

AWS SNS gateway consumes messages from SQS queue.

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// AccessKey refers K8 secret containing aws access key
AccessKey *corev1.SecretKeySelector `json:"accessKey"`

// SecretKey refers K8 secret containing aws secret key
SecretKey *corev1.SecretKeySelector `json:"secretKey"`

// Region is AWS region
Region string `json:"region"`

// Queue is AWS SQS queue to listen to for messages
Queue string `json:"queue"`

// WaitTimeSeconds is The duration (in seconds) for which the call waits for a message to arrive
// in the queue before returning.
WaitTimeSeconds int64 `json:"waitTimeSeconds"`
```

Because SQS works on polling, you need to provide a `waitTimeSeconds`.

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-sqs-gateway-configmap
data:
  notification_1: |-
    accessKey:
      name: aws-secret
      key: access
    secretKey:
      name aws-secret
      key: secret
    region: "us-east-1"
    queue: "<queue-name>"
    waitTimeSeconds: 50
```

## Setup

**1. Install Gateway**
```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/aws-sqs.yaml
```

Make sure gateway pod and service is running

**2. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/aws-sqs-gateway-configmap.yaml
```

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/aws-sqs.yaml
```

Make sure sensor pod is created.

**4. Trigger Workflow**

As soon as there a message is consumed from SQS queue, a workflow will be triggered.
