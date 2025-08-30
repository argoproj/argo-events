# Kubernetes Object Trigger

Apart from Argo workflow objects, the sensor lets you trigger any Kubernetes objects including Custom Resources
such as Pod, Deployment, Job, CronJob, etc.
Having the ability to trigger Kubernetes objects is quite powerful as providing an avenue to
set up event-driven pipelines for existing workloads.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/k8s-trigger.png?raw=true" alt="K8s Trigger"/>
</p>

<br/>
<br/>

## Trigger a K8s Pod

1.  We will use webhook event-source and sensor to trigger a K8s pod.

1.  Lets set up a webhook event source to process incoming requests.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

1.  To trigger a pod, we need to create a sensor as defined below.

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: webhook
        spec:
          template:
            serviceAccountName: create-pod-sa # A service account has privileges to create a Pod
          dependencies:
            - name: test-dep
              eventSourceName: webhook
              eventName: example
          triggers:
            - template:
                name: webhook-pod-trigger
                k8s:
                  operation: create
                  source:
                    resource:
                      apiVersion: v1
                      kind: Pod
                      metadata:
                        generateName: hello-world-
                      spec:
                        containers:
                          - name: hello-container
                            args:
                              - "hello-world"
                            command:
                              - echo
                            image: busybox
                  parameters:
                    - src:
                        dependencyName: test-dep
                      dest: spec.containers.0.args.0

1.  Create the sensor.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/trigger-standard-k8s-resource.yaml

1.  Lets expose the webhook event-source pod using `port-forward` so that we can make a request to it.

        kubectl -n argo-events port-forward <name-of-event-source-pod> 12000:12000

1.  Use either Curl or Postman to send a post request to the `http://localhost:12000/example`.

        curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1.  After the pod was completed, inspect the logs of the pod, you will see something similar as below.

        {
          "context": {
            "type": "webhook",
            "specVersion": "0.3",
            "source": "webhook",
            "eventID": "30306463666539362d346666642d343336332d383861312d336538363333613564313932",
            "time": "2020-01-11T21:23:07.682961Z",
            "dataContentType": "application/json",
            "subject": "example"
          },
          "data": "eyJoZWFkZXIiOnsiQWNjZXB0IjpbIiovKiJdLCJDb250ZW50LUxlbmd0aCI6WyIxOSJdLCJDb250ZW50LVR5cGUiOlsiYXBwbGljYXRpb24vanNvbiJdLCJVc2VyLUFnZW50IjpbImN1cmwvNy41NC4wIl19LCJib2R5Ijp7Im1lc3NhZ2UiOiJoZXkhISJ9fQ=="
        }
                
        

## Operation

You can specify the operation for the trigger using the `operation` key under triggers->template->k8s.

Operation can be either.

1. `create`: Creates the object if not available in K8s cluster.
2. `update`: Updates the object.
3. `patch`: Patches the object using given patch strategy.
4. `delete`: Deletes the object if it exists.

More info available at [here](../../APIs.md#argoproj.io/v1alpha1.StandardK8STrigger).

## Parameterization

Similar to other type of triggers, sensor offers parameterization for the K8s trigger. Parameterization is specially useful when
you want to define a generic trigger template in the sensor and populate the K8s object values on the fly.

You can learn more about trigger parameterization [here](https://argoproj.github.io/argo-events/tutorials/02-parameterization/).

## Policy

Trigger policy helps you determine the status of the triggered K8s object and decide whether to stop or continue sensor.

To determine whether the K8s object was successful or not, the K8s trigger provides a `Resource Labels` policy.
The `Resource Labels` holds a list of labels which are checked against the triggered K8s object to determine the status of the object.

                # Policy to configure backoff and execution criteria for the trigger
                # Because the sensor is able to trigger any K8s resource, it determines the resource state by looking at the resource's labels.
                policy:
                  k8s:
                    # Backoff before checking the resource labels
                    backoff:
                      # Duration is the duration in nanoseconds
                      duration: 1000000000 # 1 second
                      # Duration is multiplied by factor each iteration
                      factor: 2
                      # The amount of jitter applied each iteration
                      jitter: 0.1
                      # Exit with error after these many steps
                      steps: 5
                    # labels set on the resource decide if the resource has transitioned into the success state.
                    labels:
                      workflows.argoproj.io/phase: Succeeded
                    # Determines whether trigger should be marked as failed if the backoff times out and sensor is still unable to decide the state of the trigger.
                    # defaults to false
                    errorOnBackoffTimeout: true

Complete example is available [here](https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/trigger-with-policy.yaml).
