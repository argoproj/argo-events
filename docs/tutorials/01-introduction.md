# Introduction

In the tutorials, we will cover every aspect of Argo Events and demonstrate how you 
can leverage these features to build an event driven workflow pipeline. All the concepts you will learn
in this tutorial and subsequent ones can be applied to any type of event-source.

## Prerequisites
* Follow the installation guide to set up the Argo Events. 
* Make sure to configure Argo Workflow controller to listen to workflow objects
created in `argo-events` namespace.
* Make sure to read the concepts behind [eventbus](https://argoproj.github.io/argo-events/concepts/eventbus/),
[sensor](https://argoproj.github.io/argo-events/concepts/sensor/),
[event source](https://argoproj.github.io/argo-events/concepts/event_source/).

## Get Started

We are going to set up a sensor and event-source for webhook. The goal is to trigger an Argo workflow upon a HTTP Post request.

* Let' set up the eventbus,

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

* Create the webhook event source.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

* Create the webhook sensor.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/webhook.yaml
  
If the commands are executed successfully, the eventbus, event-source and sensor pods will get created. You will
also notice that a service is created for the event-source. 

* Expose the event-source pod via Ingress, OpenShift Route or port forward to consume requests over HTTP.

        kubectl -n argo-events port-forward <event-source-pod-name> 12000:12000

* Use either Curl or Postman to send a post request to the `http://localhost:12000/example`

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

* Now, you should see an Argo workflow being created.

        kubectl -n argo-events get wf

* Make sure the workflow pod ran successfully.

        _________________________________________ 
        / {"context":{"type":"webhook","specVersi \
        | on":"0.3","source":"webhook","e |
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

If you don't see the event-source and sensor pod in `argo-events` namespace,

  1. Make sure the correct Role and RoleBindings are applied to the service account
     and there are no errors in both event-source and sensor controller.
  2. Make sure the sensor controller configmap has `namespace` set to 
     `argo-events`.
  3. Check the logs of event-source and sensor controller. Make sure the controllers
     have processed the event-source and sensor objects and there are no errors.
  4. Look for any error in event-source or sensor pod.
  5. Inspect the event-source,

        kubectl -n argo-events event-source-object-name -o yaml

     Inspect the sensor,

        kubectl -n argo-events sensor-object-name -o yaml

     and look for any errors within the `Status`.
 
 6. Raise an issue on GitHub or post a question on `argo-events` slack channel.
