# Getting Started

We are going to set up a gateway, sensor and event-source for webhook. The goal is
to trigger an Argo workflow upon a HTTP Post request.

 * First, we need to setup event sources for gateway to listen.
   
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml
  
    The event-source drives the configuration required for a gateway to consume events from external sources.
   
 * Create webhook gateway, 
 
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
    
   After running above command, gateway controller will create corresponding a pod and service.
 
 * Create webhook sensor,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook.yaml
 
   Once sensor object is created, sensor controller will create corresponding pod and service. 
    
 * Once the gateway and sensor pods are running, dispatch a HTTP POST request to `/example` endpoint.
   Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.

        export WEBHOOK_SERVICE_URL=$(minikube service -n argo-events --url webhook-gateway-svc)

        echo $WEBHOOK_SERVICE_URL

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/example
 
   <b>Note</b>: 
   If you are facing an issue getting service url by running 
        
        minikube service -n argo-events --url webhook-gateway-svc
        
   You can use port forwarding to access the service as well,

        kubectl port-forward
        
   Open another terminal window and enter 
   
        kubectl port-forward -n argo-events <name_of_the_webhook_gateway_pod> 12000:12000
   
   You can now use `localhost:12000` to query webhook gateway
   
   Verify that an Argo workflow was triggered.

       kubectl -n argo-events get workflows | grep "webhook"
