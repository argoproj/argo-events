# Argo Workflow Trigger

Argo workflow is K8s custom resource which help orchestrating parallel jobs on Kubernetes.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/argo-workflow-trigger.png?raw=true" alt="Argo Workflow Trigger"/>
</p>

<br/>
<br/>

## Trigger a workflow

Note: You will need to have [Argo Workflows](https://argoproj.github.io/argo-workflows/) installed to make this work.

1. Make sure to have the eventbus deployed in the namespace.

1. We will use webhook event-source and sensor to trigger an Argo workflow.

1. Set up the `operate-workflow-sa` service account that the sensor will use

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/rbac/sensor-rbac.yaml

1. Let's set up a webhook event-source to process incoming requests.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

1. Create the sensor.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/special-workflow-trigger-shortened.yaml

1. Let's expose the webhook event-source pod using `port-forward` so that we can make a request to it.
  
        kubectl -n argo-events port-forward <name-of-event-source-pod> 12000:12000   

1. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`.

        curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. List the workflow using `argo list`.

## Parameterization

Similar to other type of triggers, sensor offers parameterization for the Argo workflow trigger. Parameterization is specially useful when
you want to define a generic trigger template in the sensor and populate the workflow object values on the fly.

You can learn more about trigger parameterization [here](https://argoproj.github.io/argo-events/tutorials/02-parameterization/).

## Policy

Trigger policy helps you determine the status of the triggered Argo workflow object and decide whether to stop or continue sensor.

Take a look at [K8s Trigger Policy](https://argoproj.github.io/argo-events/sensors/triggers/k8s-object-trigger/#policy).

## Argo CLI

In addition to the example above, you can leverage other functionalities
provided by the Argo CLI such as,

1. Submit
1. Submit --from
1. Resubmit
1. Resume
1. Retry
1. Suspend
1. Terminate
1. Stop

To make use of Argo CLI operations in `argoWorkflow` trigger template,

        argoWorkflow:
          operation: submit  # submit, submit-from, resubmit, resume, retry, suspend, terminate or stop

Complete example is available [here](https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/special-workflow-trigger.yaml).
