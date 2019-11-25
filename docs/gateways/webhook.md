# Webhook

The gateway runs one or more http servers in a pod. 

## Endpoints
Endpoints are activate or deactivated at the runtime. 
The gateway pod continuously monitors the event source configmap. If you add a new endpoint entry in the configmap, the server will register it as
an active endpoint and if you remove an endpoint entry, server will mark that endpoint as inactive.

## Why is there a service spec in gateway spec?
Because you'd probably want to expose the gateway to the outside world as gateway pod is running http servers. 
If you don't want to expose the gateway, just remove the `serviceSpec` from the gateway spec. 

## Setup

1. Create the [event source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/webhook.yaml).

2. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/webhook.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/webhook.yaml).

## Trigger Workflow

Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.

    export WEBHOOK_SERVICE_URL=$(minikube service -n argo-events --url <gateway_service_name>)
    echo $WEBHOOK_SERVICE_URL
    curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/foo


<b>Note</b>: 

1. If you are facing an issue getting service url by running `minikube service -n argo-events --url <gateway_service_name>`, you can use `kubectl port-forward`
2. Open another terminal window and enter `kubectl port-forward -n argo-events <name_of_the_webhook_gateway_pod> 9003:<port_on_which_gateway_server_is_running>`
3. You can now use `localhost:9003` to query webhook gateway
