# Webhook

The webhook gateway basically runs one or more http servers in a pod.
The endpoints that are registered on the http server are controller using gateway configmap.

## Endpoints
Endpoints are activate or deactivated at the runtime. The gateway pod
is continuously monitros the gateway configmap. If you add a new endpoint entry in configmap, the server will register it as
active endpoint and if you remove an endpoint entry, server will mark that endpoint as inactive.

## Why is there a service spec in gateway spec?
Because you probably want to expose the gateway to the outside world as gateway pod running is http server/s. 
If you don't to expose the gateway, just remove the `serviceSpec`. 

## Setup

**1. Install  [Event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/webhook.yaml)**

**2. Install [Gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/webhook.yaml)**

Make sure the gateway pod and service is created.

**3. Install [Sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/webhook.yaml)**

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
