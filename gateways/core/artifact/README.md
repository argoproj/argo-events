<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/s3.png?raw=true" alt="Minio S3"/>
</p>

<br/>


# Minio S3

Artifact gateway listens to bucket notifications from Minio S3 server. If you are interested in AWS S3 then 
read [AWS SNS Gateway]() 

## What types bucket notifications minio offers?
Read [here](https://docs.minio.io/docs/minio-bucket-notification-guide.html)

## How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
Endpoint  string                      `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`
Bucket    *S3Bucket                   `json:"bucket" protobuf:"bytes,2,opt,name=bucket"`
Region    string                      `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
Insecure  bool                        `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`
AccessKey *corev1.SecretKeySelector   `json:"accessKey" protobuf:"bytes,5,opt,name=accessKey"`
SecretKey *corev1.SecretKeySelector   `json:"secretKey" protobuf:"bytes,6,opt,name=secretKey"`
Event     minio.NotificationEventType `json:"event,omitempty" protobuf:"bytes,7,opt,name=event"`
Filter    *S3Filter                   `json:"filter,omitempty" protobuf:"bytes,8,opt,name=filter"`
```

**1. S3Bucket** 

It contains information to describe an S3 Bucket
```go
Key  string `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
```

**2. S3Filter**

It represents filters to apply to bucket nofifications for specifying constraints on objects.
```go
Prefix string `json:"prefix" protobuf:"bytes,1,opt,name=prefix"`
Suffix string `json:"suffix" protobuf:"bytes,2,opt,name=suffix"`
```

**3. corev1.SecretKeySelector**

```go
// The name of the secret in the pod's namespace to select from.
Name `json:"name" protobuf:"bytes,1,opt,name=name"`
// The key of the secret to select from.  Must be a valid secret key.
Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
```

Most of the things declared above are straightforward but `SecretKeySelector`. 

`SecretKeySelector` basically contains the information about the Kubernetes secret that 
contains the `access` and `secret` keys required to authenticate the user.

### Example
The following gateway configmap contains an event source configurations,

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-gateway-configmap
data:
  input: |-
    bucket:
      name: input
    # minio server endpoint
    endpoint: minio-service.argo-events:9000
    # events to listen
    events: 
     - s3:ObjectCreated:Put
     - s3:ObjectCreated:Copy
    # The `prefix` and `suffix` defined in the filter are applied on the key that is retrieved from bucket notification. 
    filter:
      prefix: ""
      suffix: ""
    insecure: true
    accessKey:
      key: accesskey
      name: artifacts-minio
    secretKey:
      key: secretkey
      name: artifacts-minio
``` 

## Setup

**1. Install Gateway Configmap**

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/artifact-gateway-configmap.yaml
```

**2. Install Gateway**

**Pre-requisite - create necessary buckets in Minio.**

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/artifact-http.yaml
```

Make sure the gateway pod is created
   
**3. Install Sensor**

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/artifact.yaml
```

**4. Trigger workflow**
Drop a file onto `input` bucket and monitor workflows

## How to listen to notifications from different bucket
Simple edit the gateway configmap and add new entry that contains the configuration required to listen to new bucket, save
the configmap. The gateway will now start listening to both old and new buckets. 

