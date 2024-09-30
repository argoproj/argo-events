# NSQ

NSQ event-source subscribes to nsq pub/sub notifications and helps sensor trigger the workloads.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

            {
                "context": {
                  "type": "type_of_event_source",
                  "specversion": "cloud_events_version",
                  "source": "name_of_the_event_source",
                  "id": "unique_event_id",
                  "time": "event_time",
                  "datacontenttype": "type_of_data",
                  "subject": "name_of_the_configuration_within_event_source"
                },
                "data": {
                   "body": "Body is the message data",
                   "timestamp": "timestamp of the message",
                   "nsqdAddress": "NSQDAddress is the address of the nsq host"
                }
            }

## Specification

NSQ event-source is available [here](../../APIs.md#argoproj.io/v1alpha1.NSQEventSource).

## Setup

1.  Deploy NSQ on local K8s cluster.

        apiVersion: v1
        kind: Service
        metadata:
          name: nsqlookupd
          labels:
            app: nsq
        spec:
          ports:
            - port: 4160
              targetPort: 4160
              name: tcp
            - port: 4161
              targetPort: 4161
              name: http
          clusterIP: None
          selector:
            app: nsq
            component: nsqlookupd
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: nsqd
          labels:
            app: nsq
        spec:
          ports:
            - port: 4150
              targetPort: 4150
              name: tcp
            - port: 4151
              targetPort: 4151
              name: http
          clusterIP: None
          selector:
            app: nsq
            component: nsqd
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: nsqadmin
          labels:
            app: nsq
        spec:
          ports:
            - port: 4170
              targetPort: 4170
              name: tcp
            - port: 4171
              targetPort: 4171
              name: http
          selector:
            app: nsq
            component: nsqadmin
        ---
        apiVersion: apps/v1beta1
        kind: StatefulSet
        metadata:
          name: nsqlookupd
        spec:
          serviceName: "nsqlookupd"
          replicas: 1
          updateStrategy:
            type: RollingUpdate
          template:
            metadata:
              labels:
                app: nsq
                component: nsqlookupd
            spec:
              containers:
                - name: nsqlookupd
                  image: nsqio/nsq:v1.1.0
                  imagePullPolicy: Always
                  resources:
                    requests:
                      cpu: 30m
                      memory: 64Mi
                  ports:
                    - containerPort: 4160
                      name: tcp
                    - containerPort: 4161
                      name: http
                  livenessProbe:
                    httpGet:
                      path: /ping
                      port: http
                    initialDelaySeconds: 5
                  readinessProbe:
                    httpGet:
                      path: /ping
                      port: http
                    initialDelaySeconds: 2
                  command:
                    - /nsqlookupd
              terminationGracePeriodSeconds: 5
        ---
        apiVersion: apps/v1beta1
        kind: Deployment
        metadata:
          name: nsqd
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: nsq
              component: nsqd
          template:
            metadata:
              labels:
                app: nsq
                component: nsqd
            spec:
              containers:
                - name: nsqd
                  image: nsqio/nsq:v1.1.0
                  imagePullPolicy: Always
                  resources:
                    requests:
                      cpu: 30m
                      memory: 64Mi
                  ports:
                    - containerPort: 4150
                      name: tcp
                    - containerPort: 4151
                      name: http
                  livenessProbe:
                    httpGet:
                      path: /ping
                      port: http
                    initialDelaySeconds: 5
                  readinessProbe:
                    httpGet:
                      path: /ping
                      port: http
                    initialDelaySeconds: 2
                  volumeMounts:
                    - name: datadir
                      mountPath: /data
                  command:
                    - /nsqd
                    - -data-path
                    - /data
                    - -lookupd-tcp-address
                    - nsqlookupd.argo-events.svc:4160
                    - -broadcast-address
                    - nsqd.argo-events.svc
                  env:
                    - name: HOSTNAME
                      valueFrom:
                        fieldRef:
                          fieldPath: metadata.name
              terminationGracePeriodSeconds: 5
              volumes:
                - name: datadir
                  emptyDir: {}
        ---
        apiVersion: extensions/v1beta1
        kind: Deployment
        metadata:
          name: nsqadmin
        spec:
          replicas: 1
          template:
            metadata:
              labels:
                app: nsq
                component: nsqadmin
            spec:
              containers:
                - name: nsqadmin
                  image: nsqio/nsq:v1.1.0
                  imagePullPolicy: Always
                  resources:
                    requests:
                      cpu: 30m
                      memory: 64Mi
                  ports:
                    - containerPort: 4170
                      name: tcp
                    - containerPort: 4171
                      name: http
                  livenessProbe:
                    httpGet:
                      path: /ping
                      port: http
                    initialDelaySeconds: 10
                  readinessProbe:
                    httpGet:
                      path: /ping
                      port: http
                    initialDelaySeconds: 5
                  command:
                    - /nsqadmin
                    - -lookupd-http-address
                    - nsqlookupd.argo-events.svc:4161
              terminationGracePeriodSeconds: 5

1.  Expose NSQD by kubectl `port-forward`.

         kubectl -n argo-events port-forward service/nsqd 4151:4151

1.  Create topic `hello` and channel `my-channel`.

        curl -X POST 'http://localhost:4151/topic/create?topic=hello'

        curl -X POST 'http://localhost:4151/channel/create?topic=hello&channel=my-channel'

1.  Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/nsq.yaml

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/nsq.yaml

1.  Publish a message on topic `hello` and channel `my-channel`.

        curl -d '{"message": "hello"}' 'http://localhost:4151/pub?topic=hello&channel=my-channel'

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
