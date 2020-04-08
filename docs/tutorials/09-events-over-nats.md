# Events Over NATS

Up until now, you have seen the gateway dispatch events to sensor over HTTP. In this section, we will see how to dispatch events over [NATS](https://docs.nats.io/).

## Prerequisites

1. NATS cluster must be up and running. If you are looking to set up a test cluster,

        apiVersion: v1
        kind: Service
        metadata:
          name: nats
          namespace: argo-events
          labels:
            component: nats
        spec:
          selector:
            component: nats
          type: ClusterIP
          ports:
          - name: client
            port: 4222
          - name: cluster
            port: 6222
          - name: monitor
            port: 8222
        ---
        apiVersion: apps/v1beta1
        kind: StatefulSet
        metadata:
          name: nats
          namespace: argo-events
          labels:
            component: nats
        spec:
          serviceName: nats
          replicas: 1
          template:
            metadata:
              labels:
                component: nats
            spec:
              serviceAccountName: argo-events-sa
              containers:
              - name: nats
                image: nats:latest
                ports:
                - containerPort: 4222
                  name: client
                - containerPort: 6222
                  name: cluster
                - containerPort: 8222
                  name: monitor
                livenessProbe:
                  httpGet:
                    path: /
                    port: 8222
                  initialDelaySeconds: 10
                  timeoutSeconds: 5


## Setup

1. Create a webhook gateway as below,

        apiVersion: argoproj.io/v1alpha1
        kind: Gateway
        metadata:
          name: webhook-gateway-multi-subscribers
          labels:
            # gateway controller with instanceId "argo-events" will process this gateway
            gateways.argoproj.io/gateway-controller-instanceid: argo-events
        spec:
          replica: 1
          type: webhook
          eventSourceRef:
            name: webhook-event-source
          template:
            metadata:
              name: webhook-gateway
              labels:
                gateway-name: webhook-gateway
            spec:
              containers:
                - name: gateway-client
                  image: argoproj/gateway-client:v0.13.0
                  imagePullPolicy: Always
                  command: ["/bin/gateway-client"]
                - name: webhook-events
                  image: argoproj/webhook-gateway:v0.13.0
                  imagePullPolicy: Always
                  command: ["/bin/webhook-gateway"]
              serviceAccountName: argo-events-sa
          service:
            metadata:
              name: webhook-gateway-svc
            spec:
              selector:
                gateway-name: webhook-gateway
              ports:
                - port: 12000
                  targetPort: 12000
              type: LoadBalancer
          subscribers:
            nats:
              - name: webhook-sensor
                serverURL: nats://nats.argo-events.svc:4222
                subject: webhook-events

1. Take a closer look at subscribers. Instead of `http`, the we have declared `nats` as protocol for event transmission under `subscribers`.


        subscribers:
            nats:
              - name: webhook-sensor
                serverURL: nats://nats.argo-events.svc:4222
                subject: webhook-sensor-subject

1. Each subscriber has three keys,

        1. name: a unique name for the subscriber.
        2. serverURL: NATS server URL
        3. subject: name of the NATS subject to publish events.

1. Now, create the sensor as follows,

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
                  image: argoproj/sensor:v0.13.0
                  imagePullPolicy: Always
              serviceAccountName: argo-events-sa
          dependencies:
            - name: test-dep
              gatewayName: webhook-gateway
              eventName: example
          subscription:
            nats:
              serverURL: nats://nats.argo-events.svc:4222
              subject: webhook-events
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

1. The `subscription` under sensor spec defines NATS as protocol for event processing.

        subscription:
            nats:
              serverURL: nats://nats.argo-events.svc:4222
              subject: webhook-events 

1. Create event source for webhook as follows,

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml
        
1. Expose the gateway pod via Ingress, OpenShift Route or port forward to consume requests over HTTP.

        kubectl -n argo-events port-forward <gateway-pod-name> 12000:12000

1. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. You should see an Argo workflow being created.

        kubectl -n argo-events get wf

## Events over HTTP and NATS

You can easily set up a gateway to send events over both HTTP and NATS,

        apiVersion: argoproj.io/v1alpha1
        kind: Gateway
        metadata:
          name: webhook-gateway-multi-subscribers
          labels:
            # gateway controller with instanceId "argo-events" will process this gateway
            gateways.argoproj.io/gateway-controller-instanceid: argo-events
        spec:
          replica: 1
          type: webhook
          eventSourceRef:
            name: webhook-event-source
          template:
            metadata:
              name: webhook-gateway
              labels:
                gateway-name: webhook-gateway
            spec:
              containers:
                - name: gateway-client
                  image: argoproj/gateway-client:v0.13.0
                  imagePullPolicy: Always
                  command: ["/bin/gateway-client"]
                - name: webhook-events
                  image: argoproj/webhook-gateway:v0.13.0
                  imagePullPolicy: Always
                  command: ["/bin/webhook-gateway"]
              serviceAccountName: argo-events-sa
          service:
            metadata:
              name: webhook-gateway-svc
            spec:
              selector:
                gateway-name: webhook-gateway
              ports:
                - port: 12000
                  targetPort: 12000
              type: LoadBalancer
          subscribers:
            http:
              - "http://webhook-time-filter-sensor.argo-events.svc:9300/"
            nats:
              - name: webhook-sensor
                serverURL: nats://nats.argo-events.svc:4222
                subject: webhook-events

1. The `subscribers` list both `http` and `nats` subscribers.

1. The sensor can also have both `http` and `nats` subscription as follows,

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: webhook-sensor-over-http-and-nats
          labels:
            sensors.argoproj.io/sensor-controller-instanceid: argo-events
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
            nats:
              serverURL: nats://nats.argo-events.svc:4222
              subject: webhook-events
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

1. Make sure that you don't define same subscriber in both `http` and `nats` under gateway subscribers. Otherwise, the gateway will
   end up dispatching same event twice, once over HTTP and once over NATS.
