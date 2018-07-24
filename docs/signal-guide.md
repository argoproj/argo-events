# Signal Guide
Conceptually, signals are the sensor's dependencies. In implementation, signals are separate microservice deployments that expose a `Listen()` gRPC API. To take advantage of the certain signal sources, you can follow this guide to help you get started installing these other services on your kubernetes cluster as well as the compatible microservices to transform events from these services into actionable signals.

## What is a signal?
A signal is a dependency, namely these come in 5 types:
- `Stream` - messages on a queue/topic
- `Artifact` - S3 Bucket Notifications
- `Calendar` - date/time schedules or intervals
- `Resource` - Kubernetes resources
- `Webhook` - HTTP webhook notifications (Git, JIRA, Trello etc.)

In order to take advantage of the various signal types, you may need to install compatible message platforms (e.g. amqp, mmqp, NATS, etc..) and s3 api compatible object storage servers (e.g. Minio, Rook, CEPH, NetApp). See the  [artifact guide](artifact-guide.md) for installing object stores.

## Sensor Controller
The `sensor-controller` is responsible for managing the `Sensor` resources, listening on sensor signals, and executing sensor triggers.

## Signal Deployments
Signals are configured as separate deployments to the main sensor controller. Signals are registered as stateless [micro](https://github.com/micro/go-micro) "microservices". Users can specify in the deployment spec for each signal how many replica pods to run for the particular sensor in order to increase event bandwidth available and also make signals more resilient.

## Types of Signals & their deployments

### Calendars
Time-based signals can include signals based on a [cron](https://crontab.guru/) schedule or an [interval duration](https://golang.org/pkg/time/#ParseDuration). In addition, calendar signals currently support a `recurrence` field in which to specify special exclusion dates for which this signal will not produce an event.

### Webhooks
Webhook signals exposes a basic HTTP server endpoint. Users can register a REST API endpoint. See Request Methods in RFC7231 to define the HTTP REST endpoint.

### Kubernetes Resources
Resource signals support watching Kubernetes resources. Users can specify `group`, `version`, `kind`, and filters including prefix of the object name, labels, annotations, and createdBy time.

### Artifacts
Artifact signals support S3 `bucket-notifications` via [Minio](https://docs.minio.io/docs/minio-bucket-notification-guide). Note that a supported notification target must be running, exposed, and configured in the Minio server. For more information, please refer to the [artifact guide](artifact-guide.md).

### Streams
Stream signals contain a generic specification for messages received on a queue and/or though messaging server. The following are the `builtin` supported stream signals. Users can build their own signals by adding implementations to the [custom](../signals/stream/custom/doc.go) package.

#### NATS
[Nats](https://nats.io/) is an open-sourced, lightweight, secure, and scalable messaging system for cloud native applications and microservices architecture. It is currently a hosted CNCF Project. We are currently experimenting with using NATS as a solution for signals (inputs) and triggers (outputs), however `NATS Streaming`, the data streaming system powered by NATS, offers many  additional [features](https://nats.io/documentation/streaming/nats-streaming-intro/) on top of the core NATS platform that we believe are very desirable and definite future enhancements.
```
signals:
    - name: nats-signal
      stream:
        type: NATS
        url: nats://example-nats-cluster:4222
        attributes:
            subject: hello
```


#### MQTT
[MMQP](http://mqtt.org/) is a M2M "Internet of Things" connectivity protocol (ISO/IEC PRF 20922) designed to be extremely lightweight and ideal for mobile applications. Some broker implementations can be found [here](https://github.com/mqtt/mqtt.github.io/wiki/brokers).
```
signals:
    - name: mqtt-signal
      stream:
        type: MQTT
        url: tcp://localhost:1883
        attributes:
            topic: hello
```


#### AMQP
[AMQP](https://www.amqp.org/) is a open standard messaging protocol (ISO/IEC 19464). There are a variety of broker implementations including, but not limited to the following:
- [Apache ActiveMQ](http://activemq.apache.org/)
- [Apache Qpid](https://qpid.apache.org/)
- [StormMQ](http://stormmq.com/)
- [RabbitMQ](https://www.rabbitmq.com/)
```
signals:
    - name: amqp-signal
      stream:
        type: AMQP
        url: amqp://localhost:5672
        attributes:
            exchangeName: myExchangeName
            exchangeType: fanout
            routingKey: myRoutingKey
```


#### Kafka
[Apache Kafka](https://kafka.apache.org/) is a distributed streaming platform. We use Shopify's [sarama](https://github.com/Shopify/sarama) client for consuming Kafka messages.
```
signals:
    - name: kafka-signal
      stream:
        type: KAFKA
        url: tcp://localhost:1883
        attributes:
            topic: hello
            partition: "0"
```
