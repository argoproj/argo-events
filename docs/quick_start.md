# Getting Started

We are going to set up a sensor and event-source for webhook. The goal is to trigger an Argo workflow upon an HTTP Post request.

Note: You will need to have [Argo Workflows](https://argoproj.github.io/argo-workflows/) installed to make this work.
The Argo Workflow controller will need to be configured to listen for Workflow objects created in `argo-events` namespace.
  (See [this](https://github.com/argoproj/argo-workflows/blob/master/docs/managed-namespace.md) link.)
  The Workflow Controller will need to be installed either in a cluster-scope configuration (i.e. no "--namespaced" argument) so that it has visibility to all namespaces, or with "--managed-namespace" set to define "argo-events" as a namespace it has visibility to. To deploy Argo Workflows with a cluster-scope configuration you can use this installation yaml file, setting `ARGO_WORKFLOWS_VERSION` with your desired version. A list of versions can be found by viewing [these](https://github.com/argoproj/argo-workflows/tags) project tags in the Argo Workflow GitHub repository.

        export ARGO_WORKFLOWS_VERSION=3.5.4
        kubectl create namespace argo
        kubectl apply -n argo -f https://github.com/argoproj/argo-workflows/releases/download/v$ARGO_WORKFLOWS_VERSION/install.yaml

1. Install Argo Events

        kubectl create namespace argo-events
        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml
        # Install with a validating admission controller
        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install-validating-webhook.yaml


1. Make sure to have the eventbus pods running in the namespace. Run following command to create the eventbus.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

1. Setup event-source for webhook as follows.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

   The above event-source contains a single event configuration that runs an HTTP server on port `12000` with endpoint `example`.

   After running the above command, the event-source controller will create a pod and service.

1. Create a service account with RBAC settings to allow the sensor to trigger workflows, and allow workflows to function.

         # sensor rbac
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/sensor-rbac.yaml
         # workflow rbac
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/workflow-rbac.yaml

1. Create webhook sensor.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/webhook.yaml

   Once the sensor object is created, sensor controller will create corresponding pod and a service.

1. Expose the event-source pod via Ingress, OpenShift Route or port forward to consume requests over HTTP.

        kubectl -n argo-events port-forward svc/webhook-eventsource-svc 12000:12000

1. Use either Curl or Postman to send a post request to the <http://localhost:12000/example>.

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. Verify that an Argo workflow was triggered.

        kubectl -n argo-events get workflows | grep "webhook"
