# Minio

Minio event-source listens to minio bucket notifications and helps sensor trigger the workloads.

**_Note_**: Minio event-source is exclusive for the Minio server. If you want to trigger workloads on AWS S3 bucket notification,
please set up the AWS SNS event-source.

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
              notification: [
                {
                  /* Minio notification. More info is available at https://docs.min.io/docs/minio-bucket-notification-guide.html
                }
              ]
            }
        }

## Setup

1. Make sure to have the minio server deployed and reachable from the event-source.

1. If you are running Minio locally, make sure to `port-forward` to minio pod in order to make the service available outside local K8s cluster.

        kubectl -n argo-events port-forward <minio-pod-name> 9000:9000 

1. Configure the minio client `mc`.

        mc config host add minio http://localhost:9000 minio minio123

1. Create a K8s secret that holds the access and secret key. This secret will be referred in the minio event source definition that we are going to install in a later step.

        apiVersion: v1
        data:
          # base64 of minio
          accesskey: bWluaW8=
          # base64 of minio123
          secretkey: bWluaW8xMjM=
        kind: Secret
        metadata:
          name: artifacts-minio
          namespace: argo-events

1. The event source we are going to use configures notifications for a bucket called `input`.

        mc mb minio/input

1. Let's install event source in the `argo-events` namespace.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/minio.yaml

1. Let's create the sensor.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/minio.yaml   

1. Create a file named and `hello-world.txt` and upload it onto to the `input` bucket. This will trigger the argo workflow.

1. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
