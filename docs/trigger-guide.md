# Trigger Guide
Triggers are the sensor's actions. Triggers are only executed after all of the sensor's signals have been resolved.

## What is a trigger?
A `trigger` is an output or an action, namely:
- a Kubernetes resource object
- a message on a streaming service

## Types of Trigger

### Resource Object
Resources define a YAML or JSON K8 resource. The set of currently resources supported are implemented in the `store` package. Adding support for new resources is as simple as including the type you want to create in the store's `decodeAndUnstructure()` method. 

List of currently supported K8 Resources:
- Sensor
- [Workflow](https://github.com/argoproj/argo)
- [Deployment](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#deployment-v1-apps)
- [Job](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#job-v1-batch)

### Messages
Messages define content and a queue resource on which to send the content. 

List of currently supported Streams:
- NATS
- Kafka
- AMQP (RabbitMQ)
- MQTT