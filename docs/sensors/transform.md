# Event Transformation

> Available after v1.6.0

1. Lua Script: Executes user-defined Lua script to transform the event.

2. JQ Command: Evaluates JQ command to transform the event. We use <https://github.com/itchyny/gojq> to evaluate JQ commands.

### Note

* If set, transformations are applied to the event before the filters are applied.

* Either a Lua script or a JQ command can be used for the transformation, not both.

* Only event data is available for the transformation and not the context.

* The event is discarded if the transformation fails.

## Lua Script

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: webhook
spec:
  template:
    serviceAccountName: operate-workflow-sa
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
      transform:
        script: |-
          event.body.message='updated'
          return event
  triggers:
    - template:
        name: webhook-workflow-trigger
        conditions: "test-dep"
        k8s:
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
                dataKey: body
              dest: spec.arguments.parameters.0.value
```

1. `transform.script` field  defines the Lua script that gets executed when an event is received.

2. The event data is available to Lua execution context via a global variable called `event`.

3. The above script sets the value of `body.message` field within the event data to a new value called `updated` and returns the event.

4. The type of the `event` variable is Table and the script must return a Table representing a valid JSON object.

## JQ Command

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: webhook
spec:
  template:
    serviceAccountName: operate-workflow-sa
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
      transform:
        jq: ".body.message *= 2"
  triggers:
    - template:
        name: webhook-workflow-trigger-1
        conditions: "test-dep-foo"
        k8s:
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
                dataKey: body
              dest: spec.arguments.parameters.0.value
```

1. The above script applies a JQ command `.body.message *= 2` on the event data which appends the value of `.body.message` to itself and
return the event.

2. The output of the transformation must be a valid JSON object.
