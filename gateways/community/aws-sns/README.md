<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/aws-sns.png?raw=true" alt="AWS SNS"/>
</p>

<br/>

# AWS SNS

AWS SNS gateway listens to notifications from SNS.

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Hook defines a webhook.
Hook      *common.Webhook           `json:"hook"`

// TopicArn is the AWS ARN
TopicArn  string                    `json:"topicArn"`

// K8s secret that holds the access key
AccessKey *corev1.SecretKeySelector `json:"accessKey" protobuf:"bytes,5,opt,name=accessKey"`

// K8s secret that holds the secret key
SecretKey *corev1.SecretKeySelector `json:"secretKey" protobuf:"bytes,6,opt,name=secretKey"`

// AWS Region
Region    string                    `json:"region"`
```

### Why is there webhook in the gateway?
Because one of the ways you can receive notifications from SNS is over http. So the gateway runs a http server internally.
Once you create a new gateway configmap or add a new event source entry in the configmap, the gateway will register the url of the server on AWS.
All notifications for that topic will then be dispatched by SNS over to the endpoint specified in event source.

### Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-sns-gateway-configmap
data:
  notification_1: |-
    topicArn: "AWS_TOPIC_ARN"
    hook:
     endpoint: "/"
     port: "12000"
     url: "URL_OF_SERVICE_EXPOSING_GATEWAY"
    accessKey:
      name: aws-secret
      key: access
    secretKey:
      name aws-secret
      key: secret
    region: "us-east-1"
```

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server to the outside world.

**How to get the URL for the service?**
Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

**1. Install Gateway**

We are installing gateway before creating configmap. Because we need to have the gateway pod running and a service backed by the pod, so 
that we can get the URL for the service. 

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/aws-sns.yaml
```

Make sure gateway pod and service is running

**2. Install Gateway Configmap**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/aws-sns-gateway-configmap.yaml
```

**3. Install Sensor**

```bash
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/aws-sns.yaml
```

Make sure sensor pod is created.

**4. Trigger Workflow**

As soon as there is message on your SNS topic, a workflow will be triggered.
 