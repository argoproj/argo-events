# MQTT Gateway & Sensor

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)
4. [Trigger Workflow](#trigger-workflow)

## Example event sources definition
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

Create gateway event sources

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/mqtt-gateway-configmap.yaml
```

## Install gateway
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/mqtt.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway mqtt-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/mqtt.yaml
```

## Trigger Workflow
Publish message to topic `foo`. You might find this helpful https://www.ev3dev.org/docs/tutorials/sending-and-receiving-messages-with-mqtt/