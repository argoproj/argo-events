# Guide

1. [Sensor and gateway controllers](controllers-guide.md)
2. [Learn about gateways](gateway-guide.md)
3. [Learn about sensors](sensor-guide.md)
4. [Learn about triggers](trigger-guide.md)
5. [Core Gateways Event Sources](#core-gateways-event-sources)
    1. [Webhook](#webhook)
    2. [Artifact](#artifact)
    3. [Calendar](#calendar)
    4. [Resource](#resource)
    5. [Streams](#streams)
6. [Sensor filters](#sensor-filters)
7. [Write your own gateway](custom-gateway.md)

## Core Gateways Event Sources
Event source name can be any valid string.

### Webhook
Webhook gateway listens to incoming HTTP requests. 

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: webhook-gateway-configmap
data:
  # each event source defines - 
  # the port for HTTP server    
  # endpoint to listen to
  # acceptable http method
  bar: |-    
    port: "12000"
    endpoint: "/bar"
    method: "POST"  
  foo: |-
    port: "12000"
    endpoint: "/foo"
    method: "POST"
``` 

## Artifact
Artifact gateway listens to minio bucket notifications.
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-gateway-configmap
data:  
  input: |-
    bucket: 
      name: input # name of the bucket we want to listen to
    endpoint: minio-service.argo-events:9000 # minio service endpoint
    event: s3:ObjectCreated:Put # type of event
    filter: # filter on object name if any
      prefix: ""
      suffix: ""
    insecure: true # type of minio server deployment
    accessKey: 
      key: accesskey # key within below k8 secret whose corresponding value is name of the accessKey
      name: artifacts-minio # k8 secret name that holds minio creds
    secretKey:
      key: secretkey # key within below k8 secret whose corresponding value is name of the secretKey
      name: artifacts-minio # k8 secret name that holds minio creds
``` 

## Calendar
Calendar gateway either accepts `interval` and `cron schedules` as event source. 
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: calendar-gateway-configmap
data:
  interval: |-
    interval: 10s # event is generated after every 10 seconds
  schedule: |-
    schedule: 30 * * * *  # event is generated after 30 min past every hour
```

## Resource
Resource gateway can monitor any K8 resource and any CRD. 
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: resource-gateway-configmap
data:
  successWorkflow: |-
    namespace: argo-events # namespace where wf is deloyef
    group: "argoproj.io" # wf group
    version: "v1alpha1" # wf version
    kind: "Workflow" # object kind
    filter: # filters can be applied on labels, annotations, creation time and name 
      labels:
        workflows.argoproj.io/phase: Succeeded
        name: "my-workflow"
  failureWorkflow: |-
    namespace: argo-events
    group: "argoproj.io"
    version: "v1alpha1"
    kind: "Workflow"
    filter:
      prefix: scripts-bash
      labels:
        workflows.argoproj.io/phase: Failed
```

## Streams 
 * **NATS**:
 ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: nats-gateway-configmap
    data:
      foo: |-
        url: nats://nats.argo-events:4222 # nats service
        subject: foo # subject to listen to
 ```
   
 * **KAFKA**:
 ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: kafka-gateway-configmap
    data:
      foo: |-
        url: kafka.argo-events:9092 # kafka service
        topic: foo # topic name
        partition: "0" # topic partition
      bar: |-
        url: kafka.argo-events:9092
        topic: bar
        partition: "1"
 ```
   
   
 * **MQTT**:
 ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: mqtt-gateway-configmap
    data:
      foo: |-
        url: tcp://mqtt.argo-events:1883 # mqtt service
        topic: foo # topic to listen to
      bar: |-
        url: tcp://mqtt.argo-events:1883
        topic: bar

 ```
 
 * **AMQP**
 ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: amqp-gateway-configmap
    data:
      foo: |-
        url: amqp://amqp.argo-events:5672
        exchangeName: foo
        exchangeType: fanout
        routingKey: fooK
      bar: |-
        url: amqp://amqp.argo-events:5672
        exchangeName: bar
        exchangeType: fanout
        routingKey: barK
 ```

## Sensor Filters
 Following are the types of the filter you can apply on signal/event payload,
    
 |   Type   |   Description      |
 |----------|-------------------|
 |   Time            |   Filters the signal based on time constraints     |
 |   EventContext    |   Filters metadata that provides circumstantial information about the signal.      |
 |   Data            |   Describes constraints and filters for payload      |
    
 ### Time Filter
   ```yaml 
   filters:
    time:
     start: "2016-05-10T15:04:05Z07:00"
     stop: "2020-01-02T15:04:05Z07:00"
   ```
 
 Example:  
 https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/time-filter-webhook.yaml
 
 ### EventContext Filter
  ``` 
  filters:
   context:
    source:
     host: amazon.com
     contentType: application/json
  ```
  
  Example:  
  https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/context-filter-webhook.yaml

 ### Data filter
 ```
 filters:
  data:
  - path: bucket
    type: string
    value: argo-workflow-input
 ```
  Example:  
  https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/data-filter-webhook.yaml
