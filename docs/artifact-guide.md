# Artifact Guide
`Sensors` support S3 artifact notifications. To take advantage of these artifact signals, you can use this guide to get started with installing artifact services.

## Minio
[Minio](https://www.minio.io/) is an distributed object storage server. Follow the Minio [Bucket Notification Guide](https://docs.minio.io/docs/minio-bucket-notification-guide) for help with configuring your minio server to listen and monitor for bucket event notifications. Note that you will need to setup a supported message queue for configuring your notification targets (i.e. NATS, WebHooks, Kafka, etc.). 

### Enabling bucket notifications
Once the Minio server is configured with a notification target and you have restarted the server to put the changes into effect, you now need to explicitely enable event notifications for a specified bucket. Enabling these notifications are out of scope of Argo Events since bucket notifications are a construct within Minio that exists at the `bucket` level. To avoid multiple sensors on the same S3 bucket conflicting with each other, creating, updating, and deleting Minio bucket notifications should be delegated to a separate process with knowledge of all notification targets including those outside of the Argo Events.
