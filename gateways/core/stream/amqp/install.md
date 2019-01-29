# AMQP Gateway & Sensor

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)

## Example event sources definition
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

Create gateway event sources

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/amqp-gateway-configmap.yaml
```

## Install gateway
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/amqp.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway amqp-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/amqp.yaml
```
