<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/stream.png?raw=true" alt="Stream"/>
</p>

<br/>


# Streams

A Stream Gateway basically listens to messages on a message queue.

The configuration for an event source is somewhat similar between all stream gateways. We will go through setup of NATS gateway.

## NATS

NATS gateway consumes messages by subscribing to NATS topic.

### How to define an event source in confimap?
You can construct an entry in configmap using following fields,
  
```go
// URL to connect to natsConfig cluster
URL string `json:"url"`

// Subject name
Subject string `json:"subject"`

// Backoff holds parameters applied to connection.
// +Optional
Backoff *wait.Backoff `json:"backoff,omitempty"`
```

If you don't provide an explicit `Backoff`, then gateway will use a default backoff in case of connection failure.

#### Example

The following event source contains information to subscribe to topic `foo`, it also contains an explicit backoff option
in case of connection failure

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nats-gateway-configmap
data:
  foo: |-
    url: nats://nats.argo-events:4222
    subject: foo
    backoff:
      # 10 seconds
      duration: 10
      steps: 5
      factor: 1.0
``` 

### Setup

1. Install Gateway Configmap

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/nats-gateway-configmap.yaml
```

2. Install Gateway

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/nats.yaml
```
Make sure the gateway pod is created.
   
3. Install Sensor

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/nats.yaml
```

Make sure the sensor pod is created

## Trigger Workflow

Publish message to subject `foo`. 
You might find this useful https://github.com/nats-io/go-nats/tree/master/examples/nats-pub

# Setting up other stream gateways
You can follow examples [here](../../../examples/gateways) to setup other stream gateways.
