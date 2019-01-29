# KAFKA Gateway & Sensor

Kafka topics act as event sources for gateway

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)
4. [Trigger Workflow](#trigger-workflow)

## Example event sources definition
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

Create gateway event sources

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/kafka-gateway-configmap.yaml
```

## Install gateway
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/kafka.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway kafka-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/kafka.yaml
```

## Trigger Workflow
Send a message to topic `foo` on partition `0`. You might find this useful https://kafka.apache.org/quickstart#quickstart_send 