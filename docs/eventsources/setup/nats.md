# NATS

NATS event-source listens to NATS subject notifications and helps sensor trigger the workloads.

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
              "subject": "name_of_the_nats_subject",
              "headers": "headers_of_the_nats_message",
              "body": "message_payload"
            }
        }

## Specification

NATS event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.NATSEventsSource).

## Setup

1.  Make sure to have NATS cluster deployed in the Kubernetes. If you don't have one already installed, please refer <https://github.com/nats-io/nats-operator> for details.

    NATS cluster setup for test purposes,

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

1.  Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/nats.yaml

1.  Create the sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/nats.yaml

1.  If you are running NATS on local K8s cluster, make sure to `port-forward` to pod,

        kubectl -n argo-events port-forward <nats-pod-name> 4222:4222

1.  Publish a message for the subject specified in the event source. Refer the nats example to publish a message to the subject <https://github.com/nats-io/go-nats-examples/tree/master/patterns/publish-subscribe>.

        go run main.go -s localhost foo '{"message": "hello"}'

1.  Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
