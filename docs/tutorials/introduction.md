# Tutorial

In this tutorial, we will cover every aspect of Argo Events and demonstrate how you 
can leverage these features to build an event driven workflow pipeline.

## Pre-requisite
* Follow the installation guide to set up the Argo Events. 
* Make sure to configure Argo Workflow controller to listen to workflow objects
created in `argo-events` namespace.
* Make sure to read the concepts behind [gateway](https://argoproj.github.io/argo-events/concepts/gateway/),
[sensor](https://argoproj.github.io/argo-events/concepts/sensor/),
[event source](https://argoproj.github.io/argo-events/concepts/event_source/)
and [trigger](https://argoproj.github.io/argo-events/concepts/trigger/).

## Introduction
To start off, lets set up a basic webhook gateway and sensor that listens to events over
HTTP an Argo workflow.

* Create the webhook event source.

  ```bash
  kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml
  ```
  
* Create the webhook gateway.

  ```bash
  kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
  ```

* Create the webhook sensor.

  ```bash
  kubectl -n argo-events create -f https://github.com/argoproj/argo-events/tree/master/examples/sensors
  ```
  
If the commands are executed successfully, the gateway and sensor pods will get created. You will
also notice that a service is created for both the gateway and sensor. 

* Expose the gateway pod via Ingress, OpenShift Route
or good ol' port forwarding to consume requests over HTTP.

  ```bash
  kubectl -n port-forward <gateway-pod-name> 12000:12000
  ```

* Use either Curl or Postman to send a post request to the `http://localhost:12000/example`

  ```bash
  curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
  ```

* Now, you should see an Argo workflow being created.

  ```bash
  kubectl -n argo-events get wf
  ```

* Make sure the workflow pod was successfully run and it printed the event data.

<b>Note:</b> You will see the message printed in the workflow logs contains the event context
and data with data being base64 encoded. In later sections, we will see how to extract particular key-value
from event context or data and pass it to the workflow as arguments.

## Troubleshoot

If you don't see the gateway and sensor pod in `argo-events` namespace,

   1. Check the logs of gateway and sensor controller.
   2.  
   3. Make sure gateway and sensor controller configmap has `namespace` set to 
  `argo-events`.  
