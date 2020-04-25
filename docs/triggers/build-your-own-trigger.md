# Build Your Own Trigger

Argo Events supports a variety of triggers out of box like Argo Workflow, K8s Objects, AWS Lambda, HTTP Requests etc. But sometimes
you may want to write your own logic to trigger a pipeline or create an object in K8s cluster. An example would be to trigger
TektonCD or AirFlow pipelines on GitHub events.

## Custom Trigger

In order to plug your own implementation of the trigger with Argo Events Sensor, you need to 
implement an interface that a sensor expects. In other words, you need to implement a gRPC server
for the interface and sensor acts as a gRPC client.

### Interface

The interface exposed via proto file,

        // Trigger offers services to build a custom trigger
        service Trigger {
            // FetchResource fetches the resource to be triggered.
            rpc FetchResource(FetchResourceRequest) returns (FetchResourceResponse);
            // Execute executes the requested trigger resource.
            rpc Execute(ExecuteRequest) returns (ExecuteResponse);
            // ApplyPolicy applies policies on the trigger execution result.
            rpc ApplyPolicy(ApplyPolicyRequest) returns (ApplyPolicyResponse);
        }

The complete proto file is available [here](https://github.com/argoproj/argo-events/blob/master/sensors/triggers/trigger.proto).

Let's go over the methods declared in the interface,

1. `FetchResource`: If the trigger server needs to fetch a resource from external sources like S3, Git or a URL, this is the
    place to do so. e.g. if the trigger server aims to invoke a TektonCD pipeline and the `PipelineRun` resource lives on Git, then
    trigger server can first fetch it from Git and return it back to sensor.

2. `Execute`: In this method, the trigger server executes/invokes the trigger. e.g. TektonCD pipeline resource being
    created in K8s cluster.

3. `ApplyPolicy`: This is where your trigger implementation can check whether the triggered resource transitioned into the success state.
   Depending upon the response from the trigger server, the sensor will either stop processing subsequent triggers, or it will continue to
   process them.
   

### How to define the Custom Trigger in a sensor?

Let's look following sensor,

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: webhook-sensor
          labels:
            sensors.argoproj.io/sensor-controller-instanceid: argo-events
        spec:
          template:
            spec:
              containers:
                - name: sensor
                  image: metalgearsolid/sensor:v0.15.0
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
                custom:
                  # the url of the trigger server.
                  serverURL: tekton-trigger.argo-events.svc:9000
                  # spec is map of string->string and it is sent over to trigger server.
                  # the spec can be anything you want as per your use-case, just make sure the trigger server understands the spec map.
                  spec:
                    url: "https://raw.githubusercontent.com/VaibhavPage/tekton-cd-trigger/master/example.yaml"
                  # These parameters are applied on resource fetched and returned by the trigger server.
                  # e.g. consider a trigger server which invokes TektonCD pipeline runs, then
                  # the trigger server can return a TektonCD PipelineRun resource.
                  # The parameters are then applied on that PipelineRun resource.
                  parameters:
                    - src:
                        dependencyName: test-dep
                        dataKey: body.namespace
                      dest: metadata.namespace
              # These parameters are applied on entire template body.
              # So that you can parameterize anything under `custom` key such as `serverURL`, `spec` etc.
              parameters:
                - src:
                    dependencyName: test-dep
                    dataKey: body.url
                  dest: custom.spec.url

The sensor definition should look familiar to you. The only difference is the `custom` key under `triggers -> template`.
The specification under `custom` key defines the custom trigger.

The most important fields are,

1. `serverURL`: This is the URL of the trigger gRPC server.

1. `spec`: It is a map of string -> string. The spec can be anything you want as per your use-case. The sensor sends
    the spec to trigger server, and it is upto the trigger gRPC server to interpret the spec.

1. `parameters`: The parameters override the resource that is fetched by the trigger server and returned to sensor client.
    Read more info on payload [here](https://argoproj.github.io/argo-events/tutorials/02-parameterization/).

1. `payload`: Payload to send to trigger server. Read more on payload [here](https://argoproj.github.io/argo-events/triggers/http-trigger/#request-payload).

The complete spec for the custom trigger is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#customtrigger).

## Custom Trigger in Action

In this section, we are going to implement a trigger server in Go that invokes TektonCD pipeline on webhook events. 
You can choose any other language like Java, Python. NodeJS etc. to implement a trigger server.

1. Setup a project in your favorite IDE for Go and copy the (trigger.proto)[https://github.com/argoproj/argo-events/blob/master/sensors/triggers/trigger.proto]
into your project.

1. Run following command to create a gRPC stub in Go,

         protoc --go_out=plugins=grpc:. *.proto

1. After running above command successfully, you will see a file called `trigger.pb.go` generated under your project directory.
   If you want, instead of compiling the proto file yourself, you can refer to the generated stuf [here](https://github.com/argoproj/argo-events/blob/master/sensors/triggers/trigger.pb.go)

1. 
