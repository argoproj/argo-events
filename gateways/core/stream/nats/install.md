# NATS Gateway & Sensor

NATS subjects act as event sources for gateway

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)
4. [Trigger Workflow](#trigger-workflow)

## Example event sources definition
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

Create gateway event sources

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/nats-gateway-configmap.yaml
```

## Install gateway
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/nats.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway nats-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/nats.yaml
```

## Trigger Workflow
Publish message to subject `foo`. You might find this useful https://github.com/nats-io/go-nats/tree/master/examples/nats-pub
