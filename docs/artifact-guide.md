# Artifact Guide
This is a guide for help in utilizing artifacts within Argo Events. Sensors use artifacts for Resource Object store for use in `Resource` triggers

## Inline
Inlined artifacts are included directly within the sensor resource and decoded as a string.

## S3
Amazon Simple Storage Service (S3) is a block/file/object store for the internet. The standardized [API](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html) allows storage and retrieval of data at any time from anywhere on the web. There are a number of S3 providers that include, but are not limited to:
### Minio
Argo Events uses the [minio-go](https://github.com/minio/minio-go) client for access to any Amazon S3 compatible object store. [Minio](https://www.minio.io/) is an distributed object storage server. Follow the Minio [Name Notification Guide](https://docs.minio.io/docs/minio-bucket-notification-guide) for help with configuring your minio server to listen and monitor for bucket event notifications. Note that you will need to setup a supported message queue for configuring your notification targets (i.e. NATS, WebHooks, Kafka, etc.). 

## File
Artifacts are defined in a file that is mounted via a [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) within the `sensor-controller` pod. 

## URL
Artifacts are accessed from web via RESTful API.

## Configmap
Artifact stored in Kubernetes configmap are accessed using the key.