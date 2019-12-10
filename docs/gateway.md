# Gateway

## What is a gateway?
A gateway consumes events from event sources, transforms them into the [cloudevents specification](https://github.com/cloudevents/spec) compliant events and dispatches them to sensors.

<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/gateways.png?raw=true" alt="Gateway"/>
</p>

<br/>

## Components
A gateway has two components:

 1. <b>gateway-client</b>: It creates one or more gRPC clients depending on event sources configurations, consumes events from server, transforms these events into cloudevents and dispatches them to sensors.
     
 2. <b>gateway-server</b>: It is a gRPC server that consumes events from event sources and streams them to gateway client.
 
## Core gateways

 1. **Calendar**:
    Events produced are based on either a [cron](https://crontab.guru/) schedule or an [interval duration](https://golang.org/pkg/time/#ParseDuration). In addition, calendar gateway supports a `recurrence` field in which to specify special exclusion dates for which this gateway will not produce an event.

 2. **Webhooks**:
    Webhook gateway exposes REST API endpoints. The request received on these endpoints are treated as events. See Request Methods in RFC7231 to define the HTTP REST endpoint.

 3. **Kubernetes Resources**:
    Resource gateway supports watching Kubernetes resources. Users can specify `group`, `version`, `kind`, and filters including prefix of the object name, labels, annotations, and createdBy time.

 4. **Artifacts**:
    Artifact gateway supports S3 `bucket-notifications` via [Minio](https://docs.minio.io/docs/minio-bucket-notification-guide). Note that a supported notification target must be running, exposed.

 5. **Streams**:
    Stream gateways contain a generic specification for messages received on a queue and/or though messaging server. The following are the stream gateways offered out of box: 

    1. **NATS**:
    [Nats](https://nats.io/) is an open-sourced, lightweight, secure, and scalable messaging system for cloud native applications and microservices architecture. It is currently a hosted CNCF Project.

    2. **MQTT**:
    [MMQP](http://mqtt.org/) is a M2M "Internet of Things" connectivity protocol (ISO/IEC PRF 20922) designed to be extremely lightweight and ideal for mobile applications. Some broker implementations can be found [here](https://github.com/mqtt/mqtt.github.io/wiki/brokers).

    3. **Kafka**:
    [Apache Kafka](https://kafka.apache.org/) is a distributed streaming platform. We use Shopify's [sarama](https://github.com/Shopify/sarama) client for consuming Kafka messages.

    4. **AMQP**:
    [AMQP](https://www.amqp.org/) is a open standard messaging protocol (ISO/IEC 19464). There are a variety of broker implementations including, but not limited to the following:
        - [Apache ActiveMQ](http://activemq.apache.org/)
        - [Apache Qpid](https://qpid.apache.org/)
        - [StormMQ](http://stormmq.com/)
        - [RabbitMQ](https://www.rabbitmq.com/)

 You can find core gateways [here](https://github.com/argoproj/argo-events/tree/master/gateways/core)

## Community gateways
You can find gateways built by the community [here](https://github.com/argoproj/argo-events/tree/master/gateways/community). New gateway contributions are always welcome.

## Example

    apiVersion: argoproj.io/v1alpha1
    kind: Gateway
    metadata:
      name: webhook-gateway
      labels:
        # gateway controller with instanceId "argo-events" will process this gateway
        gateways.argoproj.io/gateway-controller-instanceid: argo-events
        # gateway controller will use this label to match with it's own version
        # do not remove
        argo-events-gateway-version: v0.11
    spec:
      type: "webhook"
      eventSource: "webhook-event-source"
      processorPort: "9330"
      eventProtocol:
        type: "HTTP"
        http:
          port: "9300"
      template:
        metadata:
          name: "webhook-gateway-http"
          labels:
            gateway-name: "webhook-gateway"
        spec:
          containers:
            - name: "gateway-client"
              image: "argoproj/gateway-client"
              imagePullPolicy: "Always"
              command: ["/bin/gateway-client"]
            - name: "webhook-events"
              image: "argoproj/webhook-gateway"
              imagePullPolicy: "Always"
              command: ["/bin/webhook-gateway"]
          serviceAccountName: "argo-events-sa"
      service:
        metadata:
          name: webhook-gateway-svc
        spec:
          selector:
            gateway-name: "webhook-gateway"
          ports:
            - port: 12000
              targetPort: 12000
          type: LoadBalancer
      watchers:
        sensors:
          - name: "webhook-sensor"


The gateway `spec` has following fields:

1. `type`: Type of the gateway. This is defined by the user.

2. `eventSource`: Refers to K8s configmap that holds the list of event sources. You can use `namespace/configmap-name` syntax to refer the configmap in a different namespace. 

3. `processorPort`: This is a gateway server port. You can leave this to `9330` unless you really have to change it to a different port.

4. `eventProtocol`: Communication protocol between sensor and gateway. For more information, head over to [communication](./communication.md)

5. `template`: Defines the specification for gateway pod.

6. `service`: Specification of a K8s service to expose the gateway pod.

7. `watchers`: List of sensors to which events must be dispatched.

## Managing Event Sources
  * The event sources configurations are managed using K8s configmap. Once the gateway resource is created with the configmap reference in it's spec, it starts watching the configmap.
  The `gateway-client` sends each event source configuration to `gateway-server` over gRPC. The `gateway-server` then parses the configuration to start consuming events from 
  external event producing entity.

  * You can modify K8s configmap containing event sources configurations anytime and `gateway-client` will intelligently pick new/deleted configurations and send them over to `gateway-server` to either
  start or stop the event sources.

## How to write a custom gateway?
To implement a custom gateway, you need to create a gRPC server and implement the service defined below.
The framework code acts as a gRPC client consuming event stream from gateway server.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/custom-gateway.png?raw=true" alt="Sensor"/>
</p>

<br/>

### Proto Definition
1. The proto file is located [here](https://github.com/argoproj/argo-events/blob/master/gateways/eventing.proto) 

2. If you choose to implement the gateway in `Go`, then you can find generated client stubs [here](https://github.com/argoproj/argo-events/blob/master/gateways/eventing.pb.go)

3. To create stubs in other languages, head over to [gRPC website](https://grpc.io/)

4. Service,

        /**
        * Service for handling event sources.
        */
        service Eventing {
            // StartEventSource starts an event source and returns stream of events.
            rpc StartEventSource(EventSource) returns (stream Event);
            // ValidateEventSource validates an event source.
            rpc ValidateEventSource(EventSource) returns (ValidEventSource);
        }


### Available Environment Variables to Server
 
 | Field                           | Description                                      |
 | ------------------------------- | ------------------------------------------------ |
 | GATEWAY_NAMESPACE               | K8s namespace of the gateway                     |
 | GATEWAY_EVENT_SOURCE_CONFIG_MAP | K8s configmap containing event source            |
 | GATEWAY_NAME                    | name of the gateway                              |
 | GATEWAY_CONTROLLER_INSTANCE_ID  | gateway controller instance id                   |
 | GATEWAY_CONTROLLER_NAME         | gateway controller name                          |
 | GATEWAY_SERVER_PORT             | Port on which the gateway gRPC server should run |
 
### Implementation
 You can follow existing implementations [here](https://github.com/argoproj/argo-events/tree/master/gateways/core)
