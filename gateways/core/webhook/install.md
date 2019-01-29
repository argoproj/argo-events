# Webhook Gateway & Sensor

REST api endpoints act as event sources for gateway

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)

## Example event sources definition
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

Create gateway event sources

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/webhook-gateway-configmap.yaml
```

## Install gateway
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/webhook-http.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway webhook-gateway-http
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-http.yaml
```
