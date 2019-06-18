# Sensor
Sensors define a set of event dependencies (inputs) and triggers (outputs). 
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/sensor.png?raw=true" alt="Sensor"/>
</p>

<br/>

## What is an event dependency?
A dependency is an event the sensor is waiting to happen. It is defined as "gateway-name:event-source-name".
Also, you can use [globs](https://github.com/gobwas/glob#syntax) to catch a set of events (e.g. "gateway-name:*").

## What is a dependency group?
A dependency group is basically a group of event dependencies.

## What is a circuit?
Circuit is any arbitrary boolean logic that can be applied on dependency groups.

## What is a trigger?
Refer [Triggers](trigger.md).

## How it works?
  1. Once the sensor receives an event from gateway either over HTTP or through NATS, it validates
  the event against dependencies defined in sensor spec. If the event is expected, then it is marked as a valid
  event and the dependency is marked as resolved.
    
  2. If you haven't defined dependency groups, sensor basically waits for all dependencies to resolve
  and then kicks off triggers in sequence. If filters are defined, sensor applies the filter on target event 
  and if events pass the filters, triggers are fired.
  
  3. If you have defined dependency groups, sensor upon receiving an event evaluates the group to which the event belongs to
  and marks the group as resolved if all other event dependencies in the group are already resolved.
  
  4. Whenever a dependency group is resolved, sensor evaluates the `circuit` defined in spec. If the `circuit` resolves to true, the
  triggers are fired. Sensor always waits for `circuit` to resolve to true before firing triggers.
  
  5. You may not want to fire all the triggers defined in sensor spec. For that, sensor offers `when` switch on triggers. Basically `when` switch is way to control when to fire certain trigger depending upon which dependency group is resolved. 

  6. After sensor fires triggers, it transitions into `complete` state, increments completion counter and initializes it's state back to running and start the process all over again. Any event that is received in-between are stored on the internal queue.

  **Note**: If you don't provide dependency groups and `circuit`, sensor performs an `AND` operation on event dependencies.

## Basic Example

Lets look at a basic example,

    apiVersion: argoproj.io/v1alpha1
    kind: Sensor
    metadata:
      name: webhook-sensor
      labels:
        sensors.argoproj.io/sensor-controller-instanceid: argo-events
        # sensor controller will use this label to match with it's own version
        # do not remove
        argo-events-sensor-version: v0.10
    spec:
      template:
        spec:
          containers:
            - name: "sensor"
              image: "argoproj/sensor"
              imagePullPolicy: Always
          serviceAccountName: argo-events-sa
      dependencies:
        - name: "webhook-gateway:example"
      eventProtocol:
        type: "HTTP"
        http:
          port: "9300"
      triggers:
        - template:
            name: webhook-workflow-trigger
            group: argoproj.io
            version: v1alpha1
            kind: Workflow
            source:
              inline: |
                apiVersion: argoproj.io/v1alpha1
                kind: Workflow
                metadata:
                  generateName: webhook-
                spec:
                  entrypoint: whalesay
                  arguments:
                    parameters:
                    - name: message
                      # this is the value that should be overridden
                      value: hello world
                  templates:
                  - name: whalesay
                    inputs:
                      parameters:
                      - name: message
                    container:
                      image: docker/whalesay:latest
                      command: [cowsay]
                      args: ["{{inputs.parameters.message}}"]
          resourceParameters:
            - src:
                event: "webhook-gateway:example"
              dest: spec.arguments.parameters.0.value

i. The `spec.template.spec` defines the template for the sensor pod.

ii. The `dependencies` define list of events the sensor is expected to receive, meaning this is an AND operation.
 
iii. `eventProtocol` express the mode of communication to receive events
from gateways. 

iv. `triggers` define list of templates, each containing specification for a K8s resource and optional parameters. 

## Circuit

Now, lets look at a more complex example involving a circuit,
    
    apiVersion: argoproj.io/v1alpha1
    kind: Sensor
    metadata:
      name: webhook-sensor-http
      labels:
        sensors.argoproj.io/sensor-controller-instanceid: argo-events
        # sensor controller will use this label to match with it's own version
        # do not remove
        argo-events-sensor-version: v0.10
    spec:
      template:
        spec:
          containers:
            - name: "sensor"
              image: "argoproj/sensor"
              imagePullPolicy: Always
          serviceAccountName: argo-events-sa
      dependencies:
        - name: "webhook-gateway-http:endpoint1"
          filters:
            name: "context-filter"
            context:
              source:
                host: xyz.com
              contentType: application/json
        - name: "webhook-gateway-http:endpoint2"
        - name: "webhook-gateway-http:endpoint3"
        - name: "webhook-gateway-http:endpoint4"
          filters:
            name: "data-filter"
            data:
              - path: bucket
                type: string
                value:
                  - "argo-workflow-input"
                  - "argo-workflow-input1"
        - name: "webhook-gateway-http:endpoint5"
        - name: "webhook-gateway-http:endpoint6"
        - name: "webhook-gateway-http:endpoint7"
        - name: "webhook-gateway-http:endpoint8"
        - name: "webhook-gateway-http:endpoint9"
      dependencyGroups:
        - name: "group_1"
          dependencies:
            - "webhook-gateway-http:endpoint1"
            - "webhook-gateway-http:endpoint2"
        - name: "group_2"
          dependencies:
            - "webhook-gateway-http:endpoint3"
        - name: "group_3"
          dependencies:
            - "webhook-gateway-http:endpoint4"
            - "webhook-gateway-http:endpoint5"
        - name: "group_4"
          dependencies:
            - "webhook-gateway-http:endpoint6"
            - "webhook-gateway-http:endpoint7"
            - "webhook-gateway-http:endpoint8"
        - name: "group_5"
          dependencies:
            - "webhook-gateway-http:endpoint9"
      circuit: "group_1 || group_2 || ((group_3 || group_4) && group_5)"
      eventProtocol:
        type: "HTTP"
        http:
          port: "9300"
      triggers:
        - template:
            when:
              any:
                - "group_1"
                - "group_2"
            name: webhook-workflow-trigger
            group: argoproj.io
            version: v1alpha1
            kind: Workflow
            source:
              inline: |
                apiVersion: argoproj.io/v1alpha1
                kind: Workflow
                metadata:
                  generateName: hello-1-
                spec:
                  entrypoint: whalesay
                  arguments:
                    parameters:
                    - name: message
                      # this is the value that should be overridden
                      value: hello world
                  templates:
                  - name: whalesay
                    inputs:
                      parameters:
                      - name: message
                    container:
                      image: docker/whalesay:latest
                      command: [cowsay]
                      args: ["{{inputs.parameters.message}}"]
          resourceParameters:
            - src:
                event: "webhook-gateway-http:endpoint1"
              dest: spec.arguments.parameters.0.value
        - template:
            name: webhook-workflow-trigger-2
            when:
              all:
                - "group_5"
                - "group_4"
            group: argoproj.io
            version: v1alpha1
            kind: Workflow
            source:
              inline: |
                apiVersion: argoproj.io/v1alpha1
                kind: Workflow
                metadata:
                  generateName: hello-world-2-
                spec:
                  entrypoint: whalesay
                  templates:
                    - name: whalesay
                      container:
                        args:
                          - "hello world"
                        command:
                          - cowsay
                        image: "docker/whalesay:latest"
        - template:
            name: webhook-workflow-trigger-common
            group: argoproj.io
            version: v1alpha1
            kind: Workflow
            source:
              inline: |
                apiVersion: argoproj.io/v1alpha1
                kind: Workflow
                metadata:
                  generateName: hello-world-common-
                spec:
                  entrypoint: whalesay
                  templates:
                    - name: whalesay
                      container:
                        args:
                          - "hello world"
                        command:
                          - cowsay
                        image: "docker/whalesay:latest"

The sensor defines the list of dependencies with few containing filters. The filters are explained next. These dependencies are then grouped using `dependenciesGroups`. 

The significance of `dependenciesGroups` is, if you don't define it, the sensor will apply an `AND` operation and wait for all events to occur. But you may not always want to wait for all the specified events to occur,
but rather trigger the workflows as soon as a group or groups of event dependencies are satisfied.

To define the logic of when to trigger the workflows, `circuit` contains a boolean expression that is evaluated every time a event dependency
is satisfied. Template can optionally contain `when` switch that determines when to trigger this template. 

In the example, the first template will get triggered when either `group_1` or `group_2` dependencies groups are satisfied, the second template will get triggered only when both
`group_4` and `group_5` are triggered and the last template will be triggered every time the circuit evaluates to true.  

## Execution and Backoff Policy
    
    apiVersion: argoproj.io/v1alpha1
    kind: Sensor
    metadata:
      name: trigger-backoff
      labels:
        sensors.argoproj.io/sensor-controller-instanceid: argo-events
        # sensor controller will use this label to match with it's own version
        # do not remove
        argo-events-sensor-version: v0.10
    spec:
      template:
        spec:
          containers:
            - name: "sensor"
              image: "argoproj/sensor"
              imagePullPolicy: Always
          serviceAccountName: argo-events-sa
      dependencies:
        - name: "webhook-gateway-http:foo"
      eventProtocol:
        type: "HTTP"
        http:
          port: "9300"
      # If set to true, marks sensor state as `error` if the previous trigger round fails.
      # Once sensor state is set to `error`, no further triggers will be processed.
      errorOnFailedRound: true
      triggers:
        - template:
            name: trigger-1
            # Policy to configure backoff and execution criteria for the trigger
            # Because the sensor is able to trigger any K8s resource, it determines the resource state by looking at the resource's labels.
            policy:
              # Backoff before checking the resource labels
              backoff:
                # Duration is the duration in nanoseconds
                duration: 1000000000 # 1 second
                # Duration is multiplied by factor each iteration
                factor: 2
                # The amount of jitter applied each iteration
                jitter: 0.1
                # Exit with error after this many steps
                steps: 5
              # the criteria to decide if a resource is in success or failure state.
              # labels set on the resource decide if resource is in success or failed state.
              state:
                # Note: Set either success or failure labels. If you set both, only success labels will be considered.
    
                # Success defines labels required to identify a resource in success state
                success:
                  workflows.argoproj.io/phase: Succeeded
                # Failure defines labels required to identify a resource in failed state
                failure:
                  workflows.argoproj.io/phase: Failed
              # Determines whether trigger should be marked as failed if the backoff times out and sensor is still unable to decide the state of the trigger.
              # defaults to false
              errorOnBackoffTimeout: true
            group: argoproj.io
            version: v1alpha1
            kind: Workflow
            source:
              inline: |
                apiVersion: argoproj.io/v1alpha1
                kind: Workflow
                metadata:
                  generateName: webhook-
                spec:
                  entrypoint: whalesay
                  arguments:
                    parameters:
                    - name: message
                      # this is the value that should be overridden
                      value: hello world
                  templates:
                  - name: whalesay
                    inputs:
                      parameters:
                      - name: message
                    container:
                      image: docker/whalesay:latest
                      command: [cowsay]
                      args: ["{{inputs.parameters.message}}"]
          resourceParameters:
            - src:
                event: "webhook-gateway-http:foo"
              dest: spec.arguments.parameters.0.value
        - template:
            name: trigger-2
            policy:
              backoff:
                duration: 1000000000 # 1 second
                factor: 2
                jitter: 0.1
                steps: 5
              state:
                failure:
                  workflows.argoproj.io/phase: Failed
              errorOnBackoffTimeout: false
            group: argoproj.io
            version: v1alpha1
            kind: Workflow
            source:
              inline: |
                apiVersion: argoproj.io/v1alpha1
                kind: Workflow
                metadata:
                  generateName: hello-world-
                spec:
                  entrypoint: whalesay
                  templates:
                  - name: whalesay
                    container:
                      image: docker/whalesay:latest
                      command: [cowsay]
                      args: ["hello world"]
    
A trigger template can contain execution and backoff policy. Once the trigger is executed by template, it's
state is determined using `state` labels. If labels defined in `success` criteria matches the subset of labels defined on the
resource, the execution is treated as successful and vice-versa for labels defined in `failure` criteria. Please note that you can
only define either success or failure criteria. 

The `backoff` directs the sensor on when to check the labels of the executed trigger resource. If after the backoff retries, the sensor is not able to determine the
state of the resource, `errorOnBackoffTimeout` controls whether to mark trigger as failure.

The `errorOnFailedRound` defined outside of triggers decides whether to set the sensor state to `error` if the previous
round of triggers execution fails. 

## Filters
You can apply following filters on an event dependency. If the event payload passes the filter, then only it will
be treated as a valid event.

|   Type   |   Description      |
|----------|-------------------|
|   **Time**            |   Filters the signal based on time constraints     |
|   **EventContext**    |   Filters metadata that provides circumstantial information about the signal.      |
|   **Data**            |   Describes constraints and filters for payload      |

<br/>

### Time Filter

    filters:
            time:
              start: "2016-05-10T15:04:05Z07:00"
              stop: "2020-01-02T15:04:05Z07:00"

[Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/time-filter-webhook.yaml)

### EventContext Filter
 
    filters:
            context:
                source:
                    host: amazon.com
                contentType: application/json

[Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/context-filter-webhook.yaml)

### Data filter

    filters:
            data:
                - path: bucket
                  type: string
                  value: argo-workflow-input
    
[Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/data-filter-webhook.yaml)

## Examples
You can find sensor examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors)
