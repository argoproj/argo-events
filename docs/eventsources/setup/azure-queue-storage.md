# Azure Queue Storage

Azure Queue Storage event-source allows you to consume messages from azure storage queues.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

        {
            "context": {
              "id": "unique_event_id",
              "source": "name_of_the_event_source",
              "specversion": "cloud_events_version",
              "type": "type_of_event_source",
              "datacontenttype": "type_of_data",
              "subject": "name_of_the_configuration_within_event_source"
              "time": "event_time",
            },
            "data": {
                "messageID": "MessageID is the ID of the message",
                "body": "Body represents the message body",
                "insertionTime": "InsertionTime is the time the message was inserted into the queue",
            }
        }

## Setup

1. Create a queue called `test` either using az cli or Azure storage management console.

1. Fetch your connection string for Azure Queue Storage and base64 encode it.

1. Create a secret called `azure-secret` as follows.

        apiVersion: v1
        kind: Secret
        metadata:
          name: azure-secret
        type: Opaque
        data:
          connectionstring: <base64-connection-string>

1. Deploy the secret.

        kubectl -n argo-events apply -f azure-secret.yaml

1. Create the event source by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/azure-queue-storage.yaml

1. Inspect the event-source pod logs to make sure it was able to listen to the queue specified in the event source to consume messages.

1. Create a sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/azure-queue-storage.yaml

1. Dispatch a message to the queue.

        az storage message put -q test --content {"message": "hello"}'  --account-name mystorageaccount --connection-string "<the-connection-string>"

1. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).