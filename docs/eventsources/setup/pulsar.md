# Pulsar

Pulsar event-source subscribes to the topics, listens events and helps sensor trigger the workflows.

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
                   "body": "body is the message data",
                   "publishTime": "timestamp of the message",
                   "key": "message key"
                }
            }

## Specification

Pulsar event-source is available [here](../../APIs.md#argoproj.io/v1alpha1.PulsarEventSource).

## Setup

1.  To test locally, deploy a standalone Pulsar.

        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: pulsar
          labels:
            app: pulsar
        spec:
          replicas: 1
          template:
            metadata:
              name: pulsar
              labels:
                app: pulsar
            spec:
              containers:
                - name: pulsar
                  image: apachepulsar/pulsar:2.4.1
                  command:
                    - bin/pulsar
                    - standalone
                  imagePullPolicy: IfNotPresent
                  volumeMounts:
                    - mountPath: /pulsar/data
                      name: datadir
              restartPolicy: Always
              volumes:
                - name: datadir
                  emptyDir: {}
          selector:
            matchLabels:
              app: pulsar
        ---
        apiVersion: v1
        kind: Service
        metadata:
          name: pulsar
        spec:
          selector:
            app: pulsar
          ports:
            - port: 8080
              targetPort: 8080
              name: http
            - port: 6650
              name: another
              targetPort: 6650
          type: LoadBalancer

1.  Port forward to the pulsar pod using kubectl for port 6650.

1.  For production deployment, follow the official Pulsar documentation online.

1.  Deploy the eventsource.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/pulsar.yaml

1.  Deploy the sensor.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/pulsar.yaml

1.  Publish a message on topic `test`.

1.  Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
