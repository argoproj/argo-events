# Gateway Guide

## What is a gateway?
A gateway is a long running/repeatable process whose tasks are to process and transform either the internally produced events or 
incoming events into the cloudevents specification compliant events and dispatching them to sensors.

## Gateway Components
A gateway has two components:

 1. gateway-processor: Either generates the events internally or listens to incoming events. It then passes the events to gateway-transformer.
 The implementation of gateway-processor is provided by the user which means the user can easily create a custom gateway that has the business logic pertaining to a use-case.

 2. gateway-transformer: Transforms the incoming event from gateway-processor into a cloudevents specification compliant event. The event is then dispatched to sensors who are interested in listening to this gateway. 
 Refer https://github.com/cloudevents/spec for more info on cloudevents.


Core gateways come in 5 types:

|  Type       |               Description               | 
|-------------|-----------------------------------------|
| `Stream`    | Listens to messages on a queue/topic |
| `Artifact`  | Listens to S3 Bucket Notifications |
| `Calendar`  | Produces events internally depending upon date/time schedules or intervals |
| `Resource`  | Watches Kubernetes resources |
| `Webhook`   | Reacts to HTTP webhook notifications (Git, JIRA, Trello etc.) |


\
In order to take advantage of the various gateway types, you may need to install compatible message platforms (e.g. amqp, mmqp, NATS, etc..) and s3 api compatible object storage servers (e.g. Minio, Rook, CEPH, NetApp). See the  [artifact guide](artifact-guide.md) for installing object stores.

## Architecture
![](architecture.png) 

 
## Gateway Controller
The `gateway-controller` is responsible for managing the `Gateway` resources.

## Gateway Specification


|  Field               |  Description |
|----------------------|--------------|
| DeploySpec           | Pod specification for gateway
|  ConfigMap           | Name of the configmap containing gateway configuration/s    |
|  Type                | Type of gateway |
|  Version             | To mark event version  |
|  Service             | Name of the service to expose the gateway |
|  Sensors             | List of sensors to dispatch events to  |


## Gateway Deployment

All core gateways use kubernetes configmap to keep track of current gateway configurations. Multiple configurations can be defined for a single gateway and
each configuration will run in a separate go routine. The gateway watches updates to configmap which let us add new configuration at run time.
[Checkout core gateways specs.](https://github.com/argoproj/argo-events/tree/eventing/examples/gateways)

## How to write a custom gateway?
Follow the gateway tutorial
[Custom Gateways](custom-gateway.md)


## Types of Gateways & their configurations

###### Gateway can have zero configuration(won't be doing anything useful) to multiple configurations. A configuration can be added or removed during the runtime. 

### Calendars
Events produced can be based on a [cron](https://crontab.guru/) schedule or an [interval duration](https://golang.org/pkg/time/#ParseDuration). In addition, calendar gateway currently supports a `recurrence` field in which to specify special exclusion dates for which this gateway will not produce an event.
```
  calendar.fooConfig: |-
    interval: 10s
  calendar.barConfig: |-
    schedule: */1 * * * *
```

### Webhooks
Webhook gateway expose a basic HTTP server endpoint/s. 
Users can register multiple REST API endpoint. See Request Methods in RFC7231 to define the HTTP REST endpoint.

``` 
  # port can't be reconfigured.
  port: "12000"
  webhook.fooConfig: |-
    endpoint: "/foo"
    method: "POST"
  webhook.barConfig: |-
    endpoint: "/bar"
    method: "POST"  
```

### Kubernetes Resources
Resource gateway support watching Kubernetes resources. Users can specify `group`, `version`, `kind`, and filters including prefix of the object name, labels, annotations, and createdBy time.

``` 
  resource.fooConfig: |-
    namespace: argo-events
    group: "argoproj.io"
    version: "v1alpha1"
    kind: "Workflow"
    filter:
      prefix: scripts-bash
      labels:
        workflows.argoproj.io/phase: Succeeded
```

### Artifacts
Artifact gateway support S3 `bucket-notifications` via [Minio](https://docs.minio.io/docs/minio-bucket-notification-guide). Note that a supported notification target must be running, exposed, and configured in the Minio server. For more information, please refer to the [artifact guide](artifact-guide.md).
``` 
  s3.fooConfig: |-
    s3EventConfig:
      bucket: foo
      endpoint: minio.argo-events:9000
      event: s3:ObjectCreated:Put
      filter:
        prefix: ""
        suffix: ""
    insecure: true
    accessKey:
      key: accesskey
      name: minio
    secretKey:
      key: secretkey
      name: minio
  s3.barConfig: |-
    s3EventConfig:
      bucket: bar
      endpoint: minio.argo-events:9000
      event: s3:ObjectCreated:Get
      filter:
        prefix: "xyz"
        suffix: ""
    insecure: true
    accessKey:
      key: accesskey
      name: minio
    secretKey:
      key: secretkey
      name: minio
```

### Streams
Stream signals contain a generic specification for messages received on a queue and/or though messaging server. The following are the `builtin` supported stream signals. Users can build their own signals by adding implementations to the [custom](../signals/stream/custom/doc.go) package.

#### NATS
[Nats](https://nats.io/) is an open-sourced, lightweight, secure, and scalable messaging system for cloud native applications and microservices architecture. It is currently a hosted CNCF Project. We are currently experimenting with using NATS as a solution for gateway (inputs) and triggers (outputs), however `NATS Streaming`, the data streaming system powered by NATS, offers many  additional [features](https://nats.io/documentation/streaming/nats-streaming-intro/) on top of the core NATS platform that we believe are very desirable and definite future enhancements.
```
  nats.fooConfig: |-
    url: nats://nats.argo-events:4222
    attributes:
      subject: foo
  nats.barConfig: |-
    url: nats://nats.argo-events:4222
    attributes:
      subject: bar
```


#### MQTT
[MMQP](http://mqtt.org/) is a M2M "Internet of Things" connectivity protocol (ISO/IEC PRF 20922) designed to be extremely lightweight and ideal for mobile applications. Some broker implementations can be found [here](https://github.com/mqtt/mqtt.github.io/wiki/brokers).
```
  mqtt.fooConfig: |-
    url: tcp://mqtt.argo-events:1883
    attributes:
      topic: foo
  mqtt.barConfig: |-
    url: tcp://mqtt.argo-events:1883
    attributes:
      topic: bar
```


#### AMQP
[AMQP](https://www.amqp.org/) is a open standard messaging protocol (ISO/IEC 19464). There are a variety of broker implementations including, but not limited to the following:
- [Apache ActiveMQ](http://activemq.apache.org/)
- [Apache Qpid](https://qpid.apache.org/)
- [StormMQ](http://stormmq.com/)
- [RabbitMQ](https://www.rabbitmq.com/)
```
  amqp.fooConfig: |-
    url: amqp://amqp.argo-events:5672
    attributes:
      exchangeName: fooExchangeName
      exchangeType: fanout
      routingKey: fooRoutingKey
  amqp.barConfig: |-
    url: amqp://amqp.argo-events:5672
    attributes:
      exchangeName: barExchangeName
      exchangeType: fanout
      routingKey: barRoutingKey

```


#### Kafka
[Apache Kafka](https://kafka.apache.org/) is a distributed streaming platform. We use Shopify's [sarama](https://github.com/Shopify/sarama) client for consuming Kafka messages.
```
  kafka.fooConfig: |-
    url: kafka.argo-events:9092
    attributes:
      topic: foo
      partition: "0"
  kafka.barConfig: |-
    url: kafka.argo-events:9092
    attributes:
      topic: bar
      partition: "1"
```

### Examples
[Gateway Examples](https://github.com/argoproj/argo-events/tree/eventing/examples/gateways)
