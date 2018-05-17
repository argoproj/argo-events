# Signal Guide
Signals are the sensor's dependencies. To take advantage of the certain signal sources, you can follow this guide to help you getting started installing these other services on your kubernetes cluster. If you need to reference the underlying sensor api, please use the [api-guide](sensor-api.md).

## What is a signal?
A `signal` is a dependency, namely:
- A message on a queue
- An object created in S3
- A repeated calendar schedule
- A kubernetes resource

### Prerequisites
You need a working Kubernetes cluster at version >= 1.9. You will also need to install the `sensor-controller` into the cluster. This controller is is responsible for managing the `sensor` resources.
In order to take advantage of the various signal types, you may need to install compatible message platforms (e.g. amqp, mmqp, NATS, etc..) and s3 api compatible object storage servers (e.g. Minio, Rook, CEPH, NetApp). See the [signal guide](signal-guide.md) for more information about installing messaging platforms and [artifact guide](artifact-guide.md) for installing object stores.

## Sensor Controller
The `sensor-controller` is responsible for managing the `Sensor` resources and creating `sensor-executor` jobs. 
The following types of signals are supported:
 - Artifact signals which can include things like S3 Bucket Notifications, Git Webhooks, etc..
 - Stream signals which subscribe to messages on a queue or a topic
 - Calendar signals which contain time constraints and calendar events
 - Resource signals watch changes to Kubernetes resources

 The following types of triggers are supported:
 - Resource triggers produce Kubernetes objects
 - Message triggers produce messages on a streaming platform

## Sensor Executor
The `sensor-executor` is responsible for listening for various signals and updating the `sensor` resource with updates. There is a one-to-one mapping from `sensor` resource to executor job. Jobs are named in the following format: `{sensor-name}-sensor` and the associated job's pods are named in the following format: `{sensor-name}-sensor-{id}`. On successful resolution of a `sensor`, the job terminates and the `sensor` is marked as Successful. 


## Types of Signal Dependencies

### AMQP
[AMQP](https://www.amqp.org/) is a open standard messaging protocol (ISO/IEC 19464). There are a variety of broker implementations including, but not limited to the following:
- [Apache ActiveMQ](http://activemq.apache.org/)
- [Apache Qpid](https://qpid.apache.org/)
- [StormMQ](http://stormmq.com/)
- [RabbitMQ](https://www.rabbitmq.com/)

### MMQP
[MMQP](http://mqtt.org/) is a M2M "Internet of Things" connectivity protocol (ISO/IEC PRF 20922) designed to be extremely lightweight and ideal for mobile applications. Some broker implementations can be found [here](https://github.com/mqtt/mqtt.github.io/wiki/brokers).

### NATS
[Nats](https://nats.io/) is an open-sourced, lightweight, secure, and scalable messaging system for cloud native applications and microservices architecture. It is currently a hosted CNCF Project. We are currently experimenting with using NATS as a solution for signals (inputs) and triggers (outputs), however `NATS Streaming`, the data streaming system powered by NATS, offers many  additional [features](https://nats.io/documentation/streaming/nats-streaming-intro/) on top of the core NATS platform that we believe are very desirable and definite future enhancements.

#### NATS Operator
The `NATS` [Operator](https://github.com/nats-io/nats-operator) manages `NATS` clusters on Kubernetes. NATS clusters are implemented as CRDs and the operator automates their creation and administration.

### Install
 1. Create the natscluster.nats.io custom resource and nats operator
```
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nats-operator/master/example/deployment.yaml
```

2. Create a 1 node cluster
```
kubectl apply -f examples/natsCluster.yaml
```

