# Minio S3

Artifact gateway listens to bucket notifications from Minio S3 server. If you are interested in AWS S3 then 
read [AWS SNS Gateway](aws-sns.md) 

## Install Minio
If you dont have Minio installed already, follow this [link.](https://docs.min.io/docs/deploy-minio-on-kubernetes) 

## What types of bucket notifications minio offers?
Read about [notifications](https://docs.minio.io/docs/minio-bucket-notification-guide.html)

## Event Payload Structure
Refer [AWS S3 Notitification](https://docs.aws.amazon.com/AmazonS3/latest/dev/notification-content-structure.html)

## Setup

1. Before you setup gateway and sensor, make sure you have necessary buckets created in Minio.

2. Deploy [event source](https://github.com/argoproj/argo-events/blob/master/examples/event-sources/artifact.yaml) for the gateway. Change the 
event source configmap according to your use case.

3. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/artifact.yaml). Once the gateway pod spins up, check the logs of both `gateway-client`
 and `artifact-gateway` containers and make sure no error occurs.
   
4. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/artifact.yaml). Once the sensor pod spins up, make sure there
are no errors in sensor pod.

Drop a file onto `input` bucket and monitor workflows

## How to add new event source for a different bucket?
Simply edit the event source configmap and add new entry that contains the configuration required to listen to new bucket, save
the configmap. The gateway will now start listening to both old and new buckets. 
