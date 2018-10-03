# Tutorial

<b>Follow [getting started](https://github.com/argoproj/argo-events/blob/master/docs/quickstart.md) to setup namespace, service account and controllers</b>

## Controller
 
## Controller configmap
Provide the `instance-id` and the namespace for controller
controller configmap
e.g. 
```yaml
# The gateway-controller configmap includes configuration information for the gateway-controller
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-controller-configmap
data:
  config: |
    instanceID: argo-events  # mandatory
    namespace: my-custom-namespace # optional
```
<b>Note on `instance-id`</b>: it is used to map a gateway or sensor to a controller. 
e.g. when you create a gateway with label `gateways.argoproj.io/gateway-controller-instanceid: argo-events`, a
 controller with label `argo-events` will process that gateway. `instance-id` for controller are managed using [controller-configmap](https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-configmap.yaml)
Basically `instance-id` is used to horizontally scale controllers, so you won't end up overwhelming a controller with large
 number of gateways or sensors. Also keep in mind that `instance-id` has nothing to do with namespace where you are
 deploying controllers and gateways/sensors.


### Gateway controller
Gateway controller watches gateway resource and manages lifecycle of a gateway.
```yaml
# The gateway-controller listens for changes on the gateway CRD and creates gateway
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: gateway-controller
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: gateway-controller
    spec:
      serviceAccountName: argo-events-sa
      containers:
      - name: gateway-controller
        image: argoproj/gateway-controller:latest
        imagePullPolicy: Always
        env:
          - name: GATEWAY_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          - name: GATEWAY_CONTROLLER_CONFIG_MAP
            value: gateway-controller-configmap
```

### Sensor controller
Sensor controller watches sensor resource and manages lifecycle of a sensor.
```yaml
# The sensor sensor-controller listens for changes on the sensor CRD and creates sensor executor jobs
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: sensor-controller
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: sensor-controller
    spec:
      serviceAccountName: argo-events-sa
      containers:
      - name: sensor-controller
        image: argoproj/sensor-controller:latest
        imagePullPolicy: Always
        env:
          - name: SENSOR_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          - name: SENSOR_CONFIG_MAP
            value: sensor-controller-configmap
```

## Lets get started with gateways and sensors

#### Webhook
Create webhook gateway configmap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: webhook-gateway-configmap
data:
  # run http server on 12000
  webhook.portConfig: |-
    port: "12000"
  # listen to /bar endpoint for POST requests
  webhook.barConfig: |-
    endpoint: "/bar"
    method: "POST"
  # listen to /foo endpoint for POST requests
  webhook.fooConfig: |-
    endpoint: "/foo"
    method: "POST"
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook-gateway-configmap.yaml
```

Create webhook gateway. Make sure to create it in same namespace as configmap above.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
   # name of the gateway
  name: webhook-gateway
  labels:
    # must match with instance id of one of the gateway controllers. 
    gateways.argoproj.io/gateway-controller-instanceid: argo-events 
    gateway-name: "webhook-gateway"
spec:
  # configmap to read configurations from
  configMap: "webhook-gateway-configmap"
  # type of gateway
  type: "webhook"
  # event dispatch protocol between gateway and it's watchers
  dispatchMechanism: "HTTP"
  # version of events this gateway is generating. Required for cloudevents specification
  version: "1.0"
  # these are pod specifications
  deploySpec:
    containers:
    - name: "webhook-events"
      image: "argoproj/webhook-gateway"
      imagePullPolicy: "Always"
      command: ["/bin/webhook-gateway"]
    serviceAccountName: "argo-events-sa"
  # service specifications to expose gateway
  serviceSpec:
    selector:
      gateway-name: "webhook-gateway"
    ports:
      - port: 12000
        targetPort: 12000
    type: LoadBalancer
  # watchers are components interested in listening to events produced by this gateway
  watchers:
    sensors:
    - name: "webhook-sensor"
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
```

Check whether all configurations are running and gateway is active
```bash
kubectl get gateway webhook-gateway -o yaml
```

Create webhook sensor,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  # name of sensor
  name: webhook-sensor
  labels:
    # instance-id must match with one of the deployed sensor controller's instance-id
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  # make this sensor as long running.
  repeat: true
  serviceAccountName: argo-events-sa
  # signals/notifications this sensor is interested in.
  signals:
    # event must be from webhook-gateway and the configuration that produced this event must be
    # webhook.fooConfig
    - name: webhook-gateway/webhook.fooConfig
  triggers:
    - name: webhook-workflow-trigger
      resource:
        namespace: argo-events
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
                      args:
                        - "hello world"
                      command:
                        - cowsay
                      image: "docker/whalesay:latest"
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook.yaml
```

Send a POST request to the gateway service, and monitor namespace for new workflow
```bash
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/foo
```
```bash
argo list
```

### Passing payload from signal to trigger

#### Complete payload
Create a webhook sensor, 
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: webhook-with-resource-param-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  repeat: true
  serviceAccountName: argo-events-sa
  signals:
    - name: webhook-gateway/webhook.fooConfig
  triggers:
    - name: argo-workflow
      resource:
        namespace: argo-events
        group: argoproj.io
        version: v1alpha1
        kind: Workflow
        parameters:
          - src:
              signal: webhook-gateway/webhook.fooConfig
            # pass payload of webhook-gateway/webhook.fooConfig signal to first parameter value
            # of arguments.
            dest: spec.arguments.parameters.0.value
        source:
          inline: |
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                name: arguments-via-webhook-event
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
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/trigger-param-fix/examples/sensors/webhook-with-complete-payload.yaml
```

<b>Make sure to update webhook gateway with `webhook-with-resource-param-sensor` as it's watcher.</b>

Send a POST request to your webhook gateway,
```bash
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/foo
```

and inspect the logs of the new argo workflow using `argo logs`.

#### Filter event payload
Create a webhook sensor,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: webhook-with-resource-param-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  repeat: true
  serviceAccountName: argo-events-sa
  signals:
    - name: webhook-gateway/webhook.fooConfig
  triggers:
    - name: argo-workflow
      resource:
        namespace: argo-events
        group: argoproj.io
        version: v1alpha1
        kind: Workflow
        # The parameters from the workflow are overridden by the webhook's message
        parameters:
          - src:
              signal: webhook-gateway/webhook.fooConfig
              # extract the object corresponding to `message` key from event payload
              # of webhook-gateway/webhook.fooConfig signal
              path: message
              # if `message` key doesn't exists in event payload then default value of payload
              # passed to trigger will be `hello default`
              value: hello default
            # override the value of first parameter in arguments with above payload.
            dest: spec.arguments.parameters.0.value
        source:
          inline: |
              apiVersion: argoproj.io/v1alpha1
              kind: Workflow
              metadata:
                name: arguments-via-webhook-event
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

```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/trigger-param-fix/examples/sensors/webhook-with-resource-param.yaml
```

Post request to webhook gateway and watch new workflow being created

#### Calendar
Create configmap,
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: calendar-gateway-configmap
data:
  # generate event after every 55s
  calendar.barConfig: |-
    interval: 55s
  # generate event after every 10s
  calendar.fooConfig: |-
    interval: 10s
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/calendar-gateway-configmap.yaml
```

Create a calendar gateway,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
  name: calendar-gateway
  labels:
    gateways.argoproj.io/gateway-controller-instanceid: argo-events
    gateway-name: "calendar-gateway"
spec:
  deploySpec:
    containers:
    - name: "calendar-events"
      image: "argoproj/calendar-gateway"
      imagePullPolicy: "Always"
      command: ["/bin/calendar-gateway"]
    serviceAccountName: "argo-events-sa"
  configMap: "calendar-gateway-configmap"
  type: "calendar"
  dispatchMechanism: "HTTP"
  version: "1.0"
  watchers:
      sensors:
      - name: "calendar-sensor"
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/calendar.yaml
```

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: calendar-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  serviceAccountName: argo-events-sa
  imagePullPolicy: Always
  repeat: true
  signals:
    - name: calendar-gateway/calendar.fooConfig
  triggers:
    - name: calendar-workflow-trigger
      resource:
        namespace: argo-events
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
                  -
                    container:
                      args:
                        - "hello world"
                      command:
                        - cowsay
                      image: "docker/whalesay:latest"
                    name: whalesay
```
```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/trigger-param-fix/examples/sensors/calendar.yaml
```


Monitor your namespace for argo workflows,

```bash
argo list
```

#### Artifact
<b>Make sure to have Minio service deployed in your namespace.</b> Follow https://www.minio.io/kubernetes.html

Create configmap,
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-gateway-configmap
data:
  s3.fooConfig: |-
    s3EventConfig:
      # name of bucket on which gateway should listen for notifications
      bucket: input
      # minio service
      endpoint: minio-service.argo-events:9000
      # type of notification
      event: s3:ObjectCreated:Put
      filter:
        prefix: ""
        suffix: ""
    insecure: true
    # k8 secret that contains minio access and secret keys
    accessKey:
      key: accesskey
      name: artifacts-minio
    secretKey:
      key: secretkey
      name: artifacts-minio
```
```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/artifact-gateway-configmap.yaml
```

Create gateway,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
  name: artifact-gateway
  labels:
    gateways.argoproj.io/gateway-controller-instanceid: argo-events
    gateway-name: "artifact-gateway"
spec:
  deploySpec:
    containers:
    - name: "artifact-events"
      image: "argoproj/artifact-gateway"
      imagePullPolicy: "Always"
      command: ["/bin/artifact-gateway"]
    serviceAccountName: "argo-events-sa"
  configMap: "artifact-gateway-configmap"
  version: "1.0"
  type: "artifact"
  dispatchMechanism: "HTTP"
  watchers:
    sensors:
    - name: "artifact-sensor"
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/artifact.yaml
```

Create sensor,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: artifact-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  repeat: true
  serviceAccountName: argo-events-sa
  signals:
    - name: artifact-gateway/s3.fooConfig
  triggers:
    - name: artifact-workflow-trigger
      resource:
        namespace: argo-events
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
                  -
                    container:
                      args:
                        - "hello world"
                      command:
                        - cowsay
                      image: "docker/whalesay:latest"
                    name: whalesay
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/trigger-param-fix/examples/sensors/s3.yaml
```

Drop a file into `input` bucket and monitor namespace for argo workflow.

#### Resource
Create resource configmap,
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: resource-gateway-configmap
data:
  resource.fooConfig: |-
    namespace: argo-events
    group: "argoproj.io"
    version: "v1alpha1"
    kind: "Workflow"
    filter:
      labels:
        # watch workflows that have label `name: my-workflow`
        name: "my-workflow"
  resource.barConfig: |-
    namespace: argo-events
    group: "argoproj.io"
    version: "v1alpha1"
    kind: "Workflow"
    filter:
      # watch workflows that has phase as failed
      labels:
        workflows.argoproj.io/phase: Failed
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/resource-gateway.configmap.yaml
```

Create resource gateway,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
  name: resource-gateway
  labels:
    gateways.argoproj.io/gateway-controller-instanceid: argo-events
    gateway-name: "resource-gateway"
spec:
  deploySpec:
    containers:
    - name: "resource-events"
      image: "argoproj/resource-gateway"
      imagePullPolicy: "Always"
      command: ["/bin/resource-gateway"]
    serviceAccountName: "argo-events-sa"
  configMap: "resource-gateway-configmap"
  type: "resource"
  dispatchMechanism: "HTTP"
  version: "1.0"
  watchers:
    sensors:
    - name: "resource-sensor"
```

```bash
kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/resource.yaml
```

Create resource sensor,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: resource-sensor
  labels:
    sensors.argoproj.io/sensor-controller-instanceid: argo-events
spec:
  repeat: true
  serviceAccountName: argo-events-sa
  signals:
    - name: resource-gateway/resource.fooConfig
  triggers:
    - name: argo-workflow
      resource:
        namespace: argo-events
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
                  -
                    container:
                      args:
                        - "hello world"
                      command:
                        - cowsay
                      image: "docker/whalesay:latest"
                    name: whalesay
```

```bash
https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/resource.yaml
```

Now, create a workflow with label `name: my-workflow`,
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
  namespace: argo-events
  labels:
    name: my-workflow
spec:
  entrypoint: whalesay
  serviceAccountName: argo-events-sa
  templates:
  - container:
      args:
      - "hello world"
      command:
      - cowsay
      image: docker/whalesay:latest
    name: whalesay
```

Once workflow is created, resource sensor will trigger workflow.

#### Streams
Deploying stream gateways and sensors are pretty much same as other gateways. Just make sure you have
streaming solutions(NATS, KAFKA etc.) deployed in your namespace.
