<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/webhook.png?raw=true" alt="Webhook"/>
</p>

# Webhook

The webhook gateway basically runs one or more http servers in a pod.
The endpoints that are registered on the http server are controller using gateway configmap.

## Endpoints
Endpoints are activate or deactivated at the runtime. The gateway pod
is continuously monitros the gateway configmap. If you add a new endpoint entry in configmap, the server will register it as
active endpoint and if you remove an endpoint entry, server will mark that endpoint as inactive.

## How to define an entry in confimap?
You can construct an entry in configmap using following fields,

```go
// REST API endpoint
Endpoint string `json:"endpoint" protobuf:"bytes,1,name=endpoint"`

// Method is HTTP request method that indicates the desired action to be performed for a given resource.
// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
Method string `json:"method" protobuf:"bytes,2,name=method"`

// Port on which HTTP server is listening for incoming events.
Port string `json:"port" protobuf:"bytes,3,name=port"`

// URL is the url of the server. 
// +Optional
URL string `json:"url" protobuf:"bytes,4,name=url"`

// ServerCertPath refers the file that contains the cert.
// +Optional
ServerCertPath string `json:"serverCertPath,omitempty" protobuf:"bytes,4,opt,name=serverCertPath"`

// ServerKeyPath refers the file that contains private key
// +Optional
ServerKeyPath string `json:"serverKeyPath,omitempty" protobuf:"bytes,5,opt,name=serverKeyPath"`
```

### Example
The following configmap contains two event source configurations.

The first entry `bar` defines the endpoint `/bar`, the HTTP method that will be allowed for the endpoint and
the server port to register the endpoint on. 

The second entry `foo` defines the endpoint `/foo`, the HTTP method that will be allowed for the endpoint and
the server port to register the endpoint on. 

You would've noticed that the `port` is different in each configuration. This is because the gateway pod can 
run multiple http servers.

**But where is the `start` configuration to run the server?** 

There is no such `start` configuration that is needed explicitly to start a server.
If you define an entry that contains a port which is not defined in any previous entries in the configmap, the
gateway pod will automatically start a http server on that port.

```yaml
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: webhook-gateway-configmap
  data:
    bar: |-    
      port: "12000"
      endpoint: "/bar"
      method: "POST"  
    foo: |-
      port: "13000"
      endpoint: "/foo"
      method: "POST"
```

## Setup

**1. Install Gateway Configmap**

```yaml
 kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/webhook-gateway-configmap.yaml
```

**2. Install gateway**
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/webhook-http.yaml
```
Make sure the gateway is in active state and all the event sources are in running state.
   
**3. Install Sensor**
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-http.yaml
```

**4. Trigger Workflow**

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

## Add new endpoints
Simply edit the gateway configmap, add the new endpoint entries and save. The gateway 
will automatically register the new endpoints.

## Secure connection
If you want to have a secure server then you will need to mount the certifactes in the gateway pod and in 
the configmap entry, set the values for `serverCertPath` and `serverKeyPath` accordingly.

Example: 
1. Gateway: https://github.com/argoproj/argo-events/blob/master/examples/gateways/secure-webhook.yaml
2. Configmap: https://github.com/argoproj/argo-events/blob/master/examples/gateways/secure-webhook-gateway-configmap.yaml
