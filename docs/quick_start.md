# Getting Started

We are going to set up a gateway, sensor and event-source for webhook. The goal is
to trigger an Argo workflow upon a HTTP Post request.

Note: You will need to have [Argo Workflows](https://argoproj.github.io/docs/argo/readme.html) installed to make this work.

1. First, we need to setup event sources for gateway to listen.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

   The event-source drives the configuration required for a gateway to consume events from external sources.

1. Create webhook gateway, 

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/gateways/webhook.yaml

   After running above command, gateway controller will create corresponding a pod and service.

1. Create webhook sensor,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/webhook.yaml

   Once sensor object is created, sensor controller will create corresponding pod and service. 

1. Expose the gateway pod via Ingress, OpenShift Route or port forward to consume requests over HTTP.

        kubectl -n argo-events port-forward <gateway-pod-name> 12000:12000

1. Use either Curl or Postman to send a post request to the http://localhost:12000/example

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. Verify that an Argo workflow was triggered.

        kubectl -n argo-events get workflows | grep "webhook"
