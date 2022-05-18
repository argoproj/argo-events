# Getting Started

We are going to set up a sensor and event-source for webhook. The goal is to trigger an Argo workflow upon an HTTP Post request.

## Install Argo Workflows
Note: You will need to have [Argo Workflows](https://argoproj.github.io/argo-workflows/) installed to make this work. 
The Argo Workflow controller will need to be configured to listen for Workflow objects created in `argo-events` namespace. 
  (See [this](https://github.com/argoproj/argo-workflows/blob/master/docs/managed-namespace.md) link.) 
  The Workflow Controller will need to be installed either in a cluster-scope configuration (i.e. no "--namespaced" argument) 
  so that it has visibility to all namespaces, or with "--managed-namespace" set to define "argo-events" as a namespace it has visibility to.

  1. Install Argo workflows with a cluster-scope configuration without database and artifact repository

    kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/install.yaml

  2. Install Argo workflows with a cluster-scope configuration with database and artifact repository (for local experimentation)

    kubectl apply -n argo -f https://raw.githubusercontent.com/argoproj/argo-workflows/master/manifests/quick-start-postgres.yaml

  You will also need to set up RBAC to allow argo-workflow controller to access argo-events namespace.
  Choose one of the proposed settings:
  1. This will target only argo-events namespace:

    kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/manifests/workflow-controller-namespace-rbac.yaml

  2. This will let argo-workflows controller access all namespaces (use only for experimentation):

    kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/manifests/workflow-controller-cluster-wide-rbac.yaml

## Install argo-events bus, webhook and sensor
1. Make sure to have the eventbus pods running in the namespace. Run following command to create the eventbus.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

3. Setup event-source for webhook as follows.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

   The above event-source contains a single event configuration that runs an HTTP server on port `12000` with endpoint `example`.

   After running the above command, the event-source controller will create a pod and service.

4. Create a service account with RBAC settings to allow the sensor to trigger workflows, and allow workflows to function.
   
         # sensor rbac
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/sensor-rbac.yaml
         # workflow rbac
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/workflow-rbac.yaml

5. Create webhook sensor.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/webhook.yaml

   Once the sensor object is created, sensor controller will create corresponding pod and a service. 

6. Expose the event-source pod via Ingress, OpenShift Route or port forward to consume requests over HTTP.
      
        kubectl -n argo-events port-forward $(kubectl -n argo-events get pod -l eventsource-name=webhook -o name) 12000:12000 &

7. Use either Curl or Postman to send a post request to the http://localhost:12000/example.

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

8. Verify that an Argo workflow was triggered.

        kubectl -n argo-events get workflows | grep "webhook"
