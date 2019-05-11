# Getting Started

Lets deploy a webhook gateway and sensor,

 * First, we need to setup event sources for gateway to listen. The event sources for any gateway are managed using K8s configmap.
   
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml 
   
 * Create webhook gateway, 
 
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
    
   After running above command, gateway controller will create corresponding gateway pod and a LoadBalancing service.
 
 * Create webhook sensor,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook.yaml
    
   Once sensor resource is created, sensor controller will create corresponding sensor pod and a ClusterIP service. 
    
 * Once the gateway and sensor pods are running, trigger the webhook via a http POST request to `/example` endpoint.
   Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.

        export WEBHOOK_SERVICE_URL=$(minikube service -n argo-events --url <gateway_service_name>)
        echo $WEBHOOK_SERVICE_URL
        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/example
 
   <b>Note</b>: 
   If you are facing an issue getting service url by running 
        
        minikube service -n argo-events --url <gateway_service_name>
        
   You can use port forwarding to access the service

        kubectl port-forward
        
   Open another terminal window and enter 
   
        kubectl port-forward -n argo-events <name_of_the_webhook_gateway_pod> 9003:<port_on_which_gateway_server_is_running>
   
   You can now use `localhost:9003` to query webhook gateway
   
   Verify that the Argo workflow was run when the trigger was executed.

       argo list -n argo-events
