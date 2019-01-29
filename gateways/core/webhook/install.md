# Webhook Gateway & Sensor

REST api endpoints act as event sources for gateway

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)
4. [Trigger Workflow](#trigger-workflow)

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

## Trigger Workflow
Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.
```
export WEBHOOK_SERVICE_URL=$(minikube service -n argo-events --url <gateway_service_name>)
echo $WEBHOOK_SERVICE_URL
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/foo
```

<b>Note</b>: 
   * If you are facing an issue getting service url by running `minikube service -n argo-events --url <gateway_service_name>`, you can use `kubectl port-forward`
   * Open another terminal window and enter `kubectl port-forward -n argo-events <name_of_the_webhook_gateway_pod> 9003:<port_on_which_gateway_server_is_running>`
   * You can now use `localhost:9003` to query webhook gateway
