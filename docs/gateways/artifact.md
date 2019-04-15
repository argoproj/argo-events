# Minio S3

Artifact gateway listens to bucket notifications from Minio S3 server. If you are interested in AWS S3 then 
read [AWS SNS Gateway](../../community/aws-sns/README.md) 

## What types of bucket notifications minio offers?
Read about [notifications](https://docs.minio.io/docs/minio-bucket-notification-guide.html)

## How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/pkg/apis/common/s3.go#L26-L33).

Most of the things declared are straightforward but `SecretKeySelector`. 

`SecretKeySelector` basically contains the information about the Kubernetes secret that 
contains the `access` and `secret` keys required to authenticate the user.

### Event Payload Structure
Refer AWS S3 Notitification - https://docs.aws.amazon.com/AmazonS3/latest/dev/notification-content-structure.html

## Setup

**Pre-requisite** - Create necessary bucket/s in Minio.

**1. Install [Gateway Configmap](../../examples/gateways/artifact-gateway-configmap.yaml)**

**2. Install [Gateway](../../examples/gateways/artifact.yaml)**

Make sure the gateway pod is created
   
**3. Install [Sensor](../../examples/sensors/artifact.yaml)**

**4. Trigger workflow**

Drop a file onto `input` bucket and monitor workflows

## How to listen to notifications from different bucket
Simply edit the gateway configmap and add new entry that contains the configuration required to listen to new bucket, save
the configmap. The gateway will now start listening to both old and new buckets. 
