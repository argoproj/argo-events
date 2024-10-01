# Emitter

Emitter event-source subscribes to a channel and helps sensor trigger the workloads.

# Event Structure

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
              "topic": "name_of_the_topic",
              "body": "message_payload"
            }
        }

## Specification

Emitter event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.EmitterEventSource).

## Setup

1.  Deploy the emitter in your local K8s cluster.

        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: broker
          labels:
            app: broker
        spec:
          clusterIP: None
          ports:
            - port: 4000
              targetPort: 4000
          selector:
            app: broker
        ---
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: broker
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: broker
          template:
            metadata:
              labels:
                app: broker
            spec:
              containers:
                - env:
                    - name: EMITTER_LICENSE
                      value: "zT83oDV0DWY5_JysbSTPTDr8KB0AAAAAAAAAAAAAAAI" # This is a test license, DO NOT USE IN PRODUCTION!
                    - name: EMITTER_CLUSTER_SEED
                      value: "broker"
                    - name: EMITTER_CLUSTER_ADVERTISE
                      value: "private:4000"
                  name: broker
                  image: emitter/server:latest
                  ports:
                    - containerPort: 8080
                    - containerPort: 443
                    - containerPort: 4000
                  volumeMounts:
                    - name: broker-volume
                      mountPath: /data
              volumes:
                - name: broker-volume
                  hostPath:
                    path: /emitter #directory on host

1.  Create the event-source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/emitter.yaml

1.  Inspect the event-source pod logs to make sure it was able to subscribe to the topic specified in the event source to consume messages.

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/emitter.yaml

1.  Send a message on emitter channel using one of the clients <https://emitter.io/develop/golang/>.

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
