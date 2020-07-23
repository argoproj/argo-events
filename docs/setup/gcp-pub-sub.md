# GCP PubSub

GCP PubSub event-source subscribes to messages published by GCP publisher and helps sensor trigger workloads.

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
                    "id": "message id",
                    // Attributes represents the key-value pairs the current message
                    // is labelled with.
                    "attributes": "key-values",
                    "publishTime": "// The time at which the message was published",
                  	"body": "body refers to the message data",
                }
            }

## Specification

GCP PubSub event-source specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/event-source.md#pubsubeventsource).

## Setup

1. Fetch the project credentials JSON file from GCP console.

1. Create a K8s secret called `gcp-credentials` to store the credentials file

        apiVersion: v1
        data:
          key.json: <YOUR_CREDENTIALS_STRING_FROM_JSON_FILE>
        kind: Secret
        metadata:
          name: gcp-credentials
          namespace: argo-events
        type: Opaque

1. Create the event source by running the following command.
   
           kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/gcp-pubsub.yaml

1. Inspect the event-source pod logs to make sure it was able to subscribe to the topic.

1. Create the sensor by running the following command,
   
           kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/gcp-pubsub.yaml

1. Publish a message from GCP PubSub console.

1. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
