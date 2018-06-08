# Trigger Guide
Triggers are the sensor's actions. Triggers are only executed after all of the sensor's signals have been resolved.

## What is a trigger?
A `trigger` is an output or an action, namely:
- a Kubernetes resource object
- a message on a streaming service

## Types of Trigger

### Resource Object
Resources define a YAML or JSON K8 resource. The set of currently resources supported are implemented in the `store` package. Adding support for new resources is as simple as including the type you want to create in the store's `decodeAndUnstructure()` method. We hope to change this functionality so that permissions for CRUD operations against certain resources can be controlled through RBAC roles instead.

List of currently supported K8 Resources:
- Sensor
- [Workflow](https://github.com/argoproj/argo)

### Messages
Messages define content and a queue resource on which to send the content. 

List of currently supported Streams:
- NATS
- Kafka
- AMQP (RabbitMQ)
- MQTT