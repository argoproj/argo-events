# Introduction

In the tutorials, we will cover every aspect of Argo Events and demonstrate how you 
can leverage these features to build an event driven workflow pipeline. All the concepts you will learn
in this tutorial and subsequent ones can be applied to any type of gateway.

## Prerequisites
* Follow the installation guide to set up the Argo Events. 
* Make sure to configure Argo Workflow controller to listen to workflow objects
created in `argo-events` namespace.
* Make sure to read the concepts behind [gateway](https://argoproj.github.io/argo-events/concepts/gateway/),
[sensor](https://argoproj.github.io/argo-events/concepts/sensor/),
[event source](https://argoproj.github.io/argo-events/concepts/event_source/).

## Get Started
Lets set up a basic webhook gateway and sensor that listens to events over
HTTP and executes an Argo workflow.

* Create the webhook event source.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml
  
  
* Create the webhook gateway.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/gateways/webhook.yaml


* Create the webhook sensor.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/webhook.yaml
  
If the commands are executed successfully, the gateway and sensor pods will get created. You will
also notice that a service is created for both the gateway and sensor. 

* Expose the gateway pod via Ingress, OpenShift Route
or port forward to consume requests over HTTP.

        kubectl -n argo-events port-forward <gateway-pod-name> 12000:12000

* Use either Curl or Postman to send a post request to the `http://localhost:12000/example`

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

* Now, you should see an Argo workflow being created.

        kubectl -n argo-events get wf

* Make sure the workflow pod ran successfully.

        _________________________________________ 
        / {"context":{"type":"webhook","specVersi \
        | on":"0.3","source":"webhook-gateway","e |
        | ventID":"38376665363064642d343336352d34 |
        | 3035372d393766662d366234326130656232343 |
        | 337","time":"2020-01-11T16:55:42.996636 |
        | Z","dataContentType":"application/json" |
        | ,"subject":"example"},"data":"eyJoZWFkZ |
        | XIiOnsiQWNjZXB0IjpbIiovKiJdLCJDb250ZW50 |
        | LUxlbmd0aCI6WyIzOCJdLCJDb250ZW50LVR5cGU |
        | iOlsiYXBwbGljYXRpb24vanNvbiJdLCJVc2VyLU |
        | FnZW50IjpbImN1cmwvNy41NC4wIl19LCJib2R5I |
        | jp7Im1lc3NhZ2UiOiJ0aGlzIGlzIG15IGZpcnN0 |
        \ IHdlYmhvb2sifX0="}                      /
         ----------------------------------------- 
            \
             \
              \     
                            ##        .            
                      ## ## ##       ==            
                   ## ## ## ##      ===            
               /""""""""""""""""___/ ===        
          ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
               \______ o          __/            
                \    \        __/             
                  \____\______/   


<b>Note:</b> You will see the message printed in the workflow logs contains both the event context
and data, with data being base64 encoded. In later sections, we will see how to extract particular key-value
from event context or data and pass it to the workflow as arguments.

## Troubleshoot

If you don't see the gateway and sensor pod in `argo-events` namespace,

  1. Make sure the correct Role and RoleBindings are applied to the service account
     and there are no errors in both gateway and sensor controller.
  2. Make sure gateway and sensor controller configmap has `namespace` set to 
     `argo-events`.
  3. Check the logs of gateway and sensor controller. Make sure the controllers
     have processed the gateway and sensor objects and there are no errors.
  4. Look for any error in gateway or sensor pod.
  5. Inspect the gateway,

        kubectl -n argo-event gateway-object-name -o yaml

     Inspect the sensor,

        kubectl -n argo-events sensor-object-name -o yaml

     and look for any errors within the `Status`.
 
 6. Raise an issue on GitHub or post a question on `argo-events` slack channel.
