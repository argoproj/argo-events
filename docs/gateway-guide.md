# Gateway Guide

## What is a gateway?
A gateway is a long running/repeatable process whose tasks are to process and transform either the internally produced events or 
external events into the [cloudevents specification](https://github.com/cloudevents/spec) compliant events and dispatch them to watchers(sensors and/or gateways).

## Gateway Components
A gateway has two components:

 1. <b>gateway-processor</b>: Either generates the events internally or listens to external events.
 The implementation of gateway-processor is provided by the user which means the user can easily create a custom gateway.

 2. <b>gateway-transformer</b>: Transforms the incoming events from gateway-processor into a cloudevents specification compliant events. 
 The event is then dispatched to watchers. 
 
 Refer <b>https://github.com/cloudevents/spec </b> for more info on cloudevents specifications.


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
|  EventVersion             | To mark event version  |
|  ImageVersion         | ImageVersion is the version for gateway components images to run |
|  ServiceSpec             | Specifications of the service to expose the gateway |
|  Watchers             | Watchers are components which are interested listening to notifications from the gateway  |
|  RPCPort             | Used to communicate between gRPC gateway client and gRPC gateway server |
|  HTTPServerPort      | Used to communicate between gateway client and server over http |
|  DispatchMechanism   | Messaging mechanism used to send events from gateway to watchers |


## Gateway Deployment

All core gateways use kubernetes configmap to keep track of current gateway configurations. Multiple configurations can be defined for a single gateway and
each configuration will run in a separate go routine. The gateway watches updates to configmap which let us add new configuration at run time.

## How to write a custom gateway?
Follow this tutorial to learn more
[Custom Gateways](custom-gateway.md)


## Gateway configurations
<b>Gateway can have zero configuration(idle) or many configurations. A configuration can be added or removed during the runtime.</b> 

### Calendars
Events produced can be based on a [cron](https://crontab.guru/) schedule or an [interval duration](https://golang.org/pkg/time/#ParseDuration). In addition, calendar gateway currently supports a `recurrence` field in which to specify special exclusion dates for which this gateway will not produce an event.
```
  calendar.fooConfig: |-
    interval: 10s
  calendar.barConfig: |-
    schedule: 30 * * * *
```

### Webhooks
Webhook gateway expose a basic HTTP server endpoint/s. 
Users can register multiple REST API endpoint. See Request Methods in RFC7231 to define the HTTP REST endpoint.

``` 
  # portConfig defines port on which http server should run. Also it serves as index route. 
  webhook.portConfig: |-
    port: "12000"
    endpoint: "/"
    method: "POST"
  webhook.fooConfig: |-
    endpoint: "/foo"
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
Stream gateways contain a generic specification for messages received on a queue and/or though messaging server. The following are the `builtin` supported stream gateways. 

#### NATS
[Nats](https://nats.io/) is an open-sourced, lightweight, secure, and scalable messaging system for cloud native applications and microservices architecture. It is currently a hosted CNCF Project.
```
  nats.fooConfig: |-
    url: nats://nats.argo-events:4222
    subject: foo      
  nats.barConfig: |-
    url: nats://nats.argo-events:4222
    subject: bar          
```


#### MQTT
[MMQP](http://mqtt.org/) is a M2M "Internet of Things" connectivity protocol (ISO/IEC PRF 20922) designed to be extremely lightweight and ideal for mobile applications. Some broker implementations can be found [here](https://github.com/mqtt/mqtt.github.io/wiki/brokers).
```
  mqtt.fooConfig: |-
    url: tcp://mqtt.argo-events:1883
    topic: foo
  mqtt.barConfig: |-
    url: tcp://mqtt.argo-events:1883
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
    exchangeName: fooExchangeName
    exchangeType: fanout
    routingKey: fooRoutingKey
  amqp.barConfig: |-
    url: amqp://amqp.argo-events:5672
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
    topic: bar
    partition: "1"
```

### Examples
Explore [Gateway Examples](https://github.com/argoproj/argo-events/tree/master/examples/gateways)
