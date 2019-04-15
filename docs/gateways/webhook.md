# Webhook

The webhook gateway basically runs one or more http servers in a pod.
The endpoints that are registered on the http server are controller using gateway configmap.

## Endpoints
Endpoints are activate or deactivated at the runtime. The gateway pod
is continuously monitros the gateway configmap. If you add a new endpoint entry in configmap, the server will register it as
active endpoint and if you remove an endpoint entry, server will mark that endpoint as inactive.

## How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/common/webhook.go#L32-L44)

### Example
The [configmap](../../examples/gateways/secure-webhook-gateway-configmap.yaml) contains two event source configurations.

The first entry `bar` defines the endpoint `/bar`, the HTTP method that will be allowed for the endpoint and
the server port to register the endpoint on. 

The second entry `foo` defines the endpoint `/foo`, the HTTP method that will be allowed for the endpoint and
the server port to register the endpoint on. 

You would've noticed that the `port` is different in each configuration. This is because the gateway pod can 
run multiple http servers.

**But where is the `start` configuration to run the server?** 

There is no such `start` configuration that is needed explicitly to start a server.
If you define an entry that contains a port which is not defined in any of the previous entries in the configmap, the
gateway pod will automatically start a http server on that port.

## Why is there a service spec in gateway spec?
Because you probably want to expose the gateway to the outside world as gateway pod running is http server/s. 
If you don't to expose the gateway, just remove the `serviceSpec`. 

## Setup

**1. Install  [Gateway Configmap](../../examples/event-sources/webhook-gateway-configmap.yaml)**

**2. Install [Gateway](../../examples/gateways/webhook.yaml)**

Make sure the gateway pod and service is created.

**3. Install [Sensor](../../examples/sensors/webhook.yaml)**

Make sure the sensor pod is created.

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
1. [Gateway](../../examples/gateways/secure-webhook.yaml)
2. [Configmap](../../examples/gateways/secure-webhook-gateway-configmap.yaml)
