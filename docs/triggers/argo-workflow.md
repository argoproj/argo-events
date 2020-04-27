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

1. We will use webhook gateway and sensor to trigger an Argo workflow.

1. Lets set up a webhook gateway and event source to process incoming requests.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
        
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml

1.  To trigger a workflow, we need to create a sensor as defined below,

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: webhook-sensor
        spec:
          template:
            spec:
              containers:
                - name: sensor
                  image: argoproj/sensor:v0.13.0
                  imagePullPolicy: Always
              serviceAccountName: argo-events-sa
          dependencies:
            - name: test-dep
              gatewayName: webhook-gateway
              eventName: example
          subscription:
            http:
              port: 9300
          triggers:
            - template:
                name: webhook-workflow-trigger
                k8s:
                  group: argoproj.io
                  version: v1alpha1
                  resource: workflows
                  operation: create
                  source:
                    resource:
                      apiVersion: argoproj.io/v1alpha1
                      kind: Workflow
                      metadata:
                        generateName: webhook-
                      spec:
                        entrypoint: whalesay
                        arguments:
                          parameters:
                          - name: message
                            # the value will get overridden by event payload from test-dep
                            value: hello world
                        templates:
                        - name: whalesay
                          serviceAccountName: argo-events-sa
                          inputs:
                            parameters:
                            - name: message
                          container:
                            image: docker/whalesay:latest
                            command: [cowsay]
                            args: ["{{inputs.parameters.message}}"]
                  parameters:
                    - src:
                        dependencyName: test-dep
                      dest: spec.arguments.parameters.0.value

1. Create the sensor,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook.yaml

1. Lets expose the webhook gateway using `port-forward` so that we can make a request to it.
  
        kubectl -n argo-events port-forward <name-of-gateway-pod> 12000:12000   

1. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`

        curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. List the workflow using `argo list`.

## Parameterization

Similar to other type of triggers, sensor offers parameterization for the Argo workflow trigger. Parameterization is specially useful when
you want to define a generic trigger template in the sensor and populate the workflow object values on the fly.

You can learn more about trigger parameterization [here](https://argoproj.github.io/argo-events/tutorials/02-parameterization/).

## Policy

Trigger policy helps you determine the status of the triggered Argo workflow object and decide whether to stop or continue sensor. 

Take a look at [K8s Trigger Policy](https://argoproj.github.io/argo-events/triggers/k8s-object-trigger/#policy).

## Argo CLI

Although the sensor defined above lets you trigger an Argo workflow, it doesn't have the ability to leverage the functionality 
provided by the Argo CLI such as,

1. Submit
2. Resubmit
3. Resume
4. Retry
5. Suspend

To make use of Argo CLI operations, The sensor provides the `argoWorkflow` trigger template,

        argoWorkflow:
          group: argoproj.io
          version: v1alpha1
          resource: workflows
          operation: submit  # submit, resubmit, resume, retry or suspend 

Complete example is available [here](https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/special-workflow-trigger.yaml).
