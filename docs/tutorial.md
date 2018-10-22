# Tutorial

<b>Follow [getting started](https://github.com/argoproj/argo-events/blob/master/docs/quickstart.md) to setup namespace, service account and controllers</b>
<br>

1. [What are sensor and gateway controllers](controllers-guide.md)
2. [Core Gateways and Sensors](#gands)
    1. [Webhook](#webhook)
    2. [Artifact](#artifact)
    3. [Calendar](#calendar)
    4. [Resource](#resource)
    5. [Streams](#streams)
        1. [Nats](#nats)
        2. [Kafka](#kafka)
        3. [MQTT](#mqtt)
        4. [AMQP](#amqp)
3. [Updating configurations dynamically](#updating-configurations)
4. [Passing payload from signal to trigger](#passing-payload-from-signal-to-trigger)
5. [Sensor Filters](#sensor-filters)
6. [Fetching Triggers](#fetching-sensor-triggers) 
7. [Writing custom gateways](custom-gateway.md)
         
## <a name="gands">Gateways and Sensors</a>

## <a name="webhook">Webhook</a>

Webhook gateway is useful when you want to listen to an incoming HTTP request and forward that event to watchers. 

1) <h5>Let's have a look at the configuration for our gateway.</h5>

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
    
    1) This configmap contains multiple configurations. First configuration describes on which port HTTP server should run. Currently, the gateway
    can only start one HTTP server and all endpoints will be registered with this server. But in future, we plan to add support to 
    spin up multiple HTTP servers and give ability to user to register endpoints to different servers.
    
    2) Second configuration describes an endpoint called `/bar` that will be registered with HTTP server. The `method` describes which HTTP method
    is allowed for a request. In this case only incoming HTTP POST requests will be accepted on `/bar`.
    
    3) Third configuration has endpoint `/foo` and accepts requests with method POST.
    
    <h5>Lets go ahead and create above configmap,</h5>
    
    ```bash
    kubectl create -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook-gateway-configmap.yaml
    ```
    
    ```bash
    # Make sure that configmap is created in `argo-events` namespace
    
    kubectl -n argo-events get configmaps webhook-gateway-configmap
    ```

2) <h5>Next step is to create the webhook gateway,<h5>

    1. Gateway definition,
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
    
    2. Run following command,    
        ```bash
        kubectl create -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
        ```
    
    3. Check all gateway configurations are in `running` state
       ```bash
        kubectl get -n argo-events gateways webhook-gateway -o yaml
        ```

3) <h5>Now its time to create webhook sensor.</h5>
    1. Sensor definition,
        
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
    
        This sensor defines only one signal called `webhook-gateway/webhook.fooConfig`, meaning, it is interested in listening
        events from `webhook.fooConfig` configuration within `webhook-gateway` gateway.
    
    2. Run following command, 
        ```bash
        kubectl create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook.yaml
        ```

    3. Check whether all sensor nodes are initialized,
        ```bash
        kubectl get -n argo-events sensors webhook-sensor   
        ```

    4. Get the service url for gateway,
        ```bash
        minikube service --url webhook-gateway-gateway-svc
        ```
    
    5. If you face issue getting service url from executing above command, you can use `kubectl port-forward`
        1. Open another terminal window and enter `kubectl port-forward <name_of_the_webhook_gateway_pod> 9003:<port_on_which_gateway_server_is_running>`
        2. You can now user `localhost:9003` to query webhook gateway

    6. Send a POST request to the gateway service, and monitor namespace for new workflow
        ```bash
        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST <WEBHOOK_SERVICE_URL>/foo
        ```
    
    7. List argo workflows
        ```bash
        argo list
        ```

<br/>

## <a name="artifact">Artifact</a>
Currently framework supports Minio S3 storage for artifact gateway but we plan to add File System and AWS/GCP S3 gateways in future.

Lets start with deploying Minio server standalone deployment. You can get the K8 deployment from https://www.minio.io/kubernetes.html
   
   1. Minio deployment, store it in `minio-deployment.yaml`
        ```yaml
        apiVersion: v1
        kind: PersistentVolumeClaim
        metadata:
          # This name uniquely identifies the PVC. Will be used in deployment below.
          name: minio-pv-claim
          labels:
            app: minio-storage-claim
        spec:
          # Read more about access modes here: http://kubernetes.io/docs/user-guide/persistent-volumes/#access-modes
          accessModes:
            - ReadWriteOnce
          resources:
            # This is the request for storage. Should be available in the cluster.
            requests:
              storage: 10Gi
          # Uncomment and add storageClass specific to your requirements below. Read more https://kubernetes.io/docs/concepts/storage/persistent-volumes/#class-1
          #storageClassName:
        ---
        apiVersion: extensions/v1beta1
        kind: Deployment
        metadata:
          # This name uniquely identifies the Deployment
          name: minio-deployment
        spec:
          strategy:
            type: Recreate
          template:
            metadata:
              labels:
                # Label is used as selector in the service.
                app: minio
            spec:
              # Refer to the PVC created earlier
              volumes:
              - name: storage
                persistentVolumeClaim:
                  # Name of the PVC created earlier
                  claimName: minio-pv-claim
              containers:
              - name: minio
                # Pulls the default Minio image from Docker Hub
                image: minio/minio
                args:
                - server
                - /storage
                env:
                # Minio access key and secret key
                - name: MINIO_ACCESS_KEY
                  value: "myaccess"
                - name: MINIO_SECRET_KEY
                  value: "mysecret"
                ports:
                - containerPort: 9000
                # Mount the volume into the pod
                volumeMounts:
                - name: storage # must match the volume name, above
                  mountPath: "/storage"
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: minio-service
        spec:
          type: LoadBalancer
          ports:
            - port: 9000
              targetPort: 9000
              protocol: TCP
          selector:
            app: minio

        ```
        
   2. Install minio,
        ```bash
        kubectl create -n argo-events -f minio-deployment.yaml 
        ``` 
    
   3. Create the configuration,
        ```yaml
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: artifact-gateway-configmap
        data:
          s3.fooConfig: |-
            s3EventConfig:
              bucket: input # name of the bucket we want to listen to
              endpoint: minio-service.argo-events:9000 # minio service endpoint
              event: s3:ObjectCreated:Put # type of event
              filter: # filter on object name if any
                prefix: ""
                suffix: ""
            insecure: true # type of minio server deployment
            accessKey: 
              key: accesskey # key within below k8 secret whose corresponding value is name of the accessKey
              name: artifacts-minio # k8 secret name that holds minio creds
            secretKey:
              key: secretkey # key within below k8 secret whose corresponding value is name of the secretKey
              name: artifacts-minio # k8 secret name that holds minio creds
        ``` 
    
        Read comments on configmap to understand more about each field in configuration

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
