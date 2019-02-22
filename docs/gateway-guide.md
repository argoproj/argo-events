# Gateway Guide

1. [What is a gateway](#what-is-a-gateway)
2. [Components](#components)
2. [Specification](#specification)
4. [Managing Event Sources](#managing-event-sources)
5. [How to write a custom gateway?](#how-to-write-a-custom-gateway)
6. [Examples](#examples)

## What is a gateway?
A gateway is a long running/repeatable process which consumes events from event sources, transform them into the [cloudevents specification](https://github.com/cloudevents/spec) compliant events and dispatch them to watchers(sensors and/or gateways).

### Components
A gateway has two components:

 1. <b>gateway-client</b>: It creates one or more gRPC clients depending on event sources, consumes events from server, transforms these events into cloudevents and dispatches them to watchers.
     Refer <b>https://github.com/cloudevents/spec </b> for more info on cloudevents specifications.
     
 2. <b>gateway-server</b>: It is a gRPC server that consumes event from event sources and dispatches to gRPC client/s created by gateway client
 
### Core gateways

 1. **Calendar**:
 Events produced can be based on a [cron](https://crontab.guru/) schedule or an [interval duration](https://golang.org/pkg/time/#ParseDuration). In addition, calendar gateway currently supports a `recurrence` field in which to specify special exclusion dates for which this gateway will not produce an event.

 2. **Webhooks**:
    Webhook gateway expose a basic HTTP server endpoint/s. 
    Users can register multiple REST API endpoint. See Request Methods in RFC7231 to define the HTTP REST endpoint.

 3. **Kubernetes Resources**:
    Resource gateway support watching Kubernetes resources. Users can specify `group`, `version`, `kind`, and filters including prefix of the object name, labels, annotations, and createdBy time.

 4. **Artifacts**:
 Artifact gateway support S3 `bucket-notifications` via [Minio](https://docs.minio.io/docs/minio-bucket-notification-guide). Note that a supported notification target must be running, exposed, and configured in the Minio server. For more information, please refer to the [artifact guide](artifact-guide.md).

 5. **Streams**:
    Stream gateways contain a generic specification for messages received on a queue and/or though messaging server. The following are the `builtin` supported stream gateways. 

     1. **NATS**:
    [Nats](https://nats.io/) is an open-sourced, lightweight, secure, and scalable messaging system for cloud native applications and microservices architecture. It is currently a hosted CNCF Project.

    2. **MQTT**:
    [MMQP](http://mqtt.org/) is a M2M "Internet of Things" connectivity protocol (ISO/IEC PRF 20922) designed to be extremely lightweight and ideal for mobile applications. Some broker implementations can be found [here](https://github.com/mqtt/mqtt.github.io/wiki/brokers).

    3. **AMQP**:
    [AMQP](https://www.amqp.org/) is a open standard messaging protocol (ISO/IEC 19464). There are a variety of broker implementations including, but not limited to the following:
    - [Apache ActiveMQ](http://activemq.apache.org/)
    - [Apache Qpid](https://qpid.apache.org/)
    - [StormMQ](http://stormmq.com/)
    - [RabbitMQ](https://www.rabbitmq.com/)

    4. **Kafka**
    [Apache Kafka](https://kafka.apache.org/) is a distributed streaming platform. We use Shopify's [sarama](https://github.com/Shopify/sarama) client for consuming Kafka messages.


### Community gateways
https://github.com/argoproj/argo-events/tree/master/gateways/community

In order to take advantage of the various gateway types, you may need to install compatible message platforms (e.g. amqp, mmqp, NATS, etc..) and s3 api compatible object storage servers (e.g. Minio, Rook, CEPH, NetApp).

## Specification
https://github.com/argoproj/argo-events/blob/master/docs/gateway-protocol.md

## Managing Event Sources
All gateways use kubernetes configmap to keep track of event sources. Multiple event source can be defined for a gateway. The gateway watches updates to configmap which let us add new event sources at run time.

## How to write a custom gateway?
Follow [Custom Gateways](custom-gateway.md)

## Examples
Explore [Gateway Examples](https://github.com/argoproj/argo-events/tree/master/examples/gateways)
