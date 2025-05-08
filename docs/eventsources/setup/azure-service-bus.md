# Azure Service Bus

Service Bus event-source allows you to consume messages from queues and topics in Azure Service Bus and helps sensor trigger workflows.

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
                "applicationProperties": "ApplicationProperties can be used to store custom metadata for a message",
                "body": "Body represents the message body",
                "contentType": "ContentType is the MIME content type",
                "correlationID": "CorrelationID is the correlation identifier",
                "enqueuedTime": "EnqueuedTime is the time when the message was enqueued",
                "messageID": "ID of the message",
                "replyTo": "ReplyTo is an application-defined value specify a reply path to the receiver of the message",
                "sequenceNumber": "SequenceNumber is a unique number assigned to a message by Service Bus",
                "subject": "Subject enables an application to indicate the purpose of the message, similar to an email subject line",
            }
        }

## Setup

1. Create a queue called `test` either using Azure CLI or Azure Service Bus management console.

1. Fetch your connection string for Azure Service Bus and base64 encode it.

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

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/azure-service-bus.yaml

1. Inspect the event-source pod logs to make sure it was able to listen to the queue specified in the event source to consume messages.

1. Create a sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/azure-service-bus.yaml

1. Lets set up a Service Bus client. If you don't have `azure-servicebus`installed, run.

        python -m pip install azure-servicebus --upgrade

1. Open a python REPL and run the following code to send a message on the queue called `test`.

    Before running the code, make sure you have the `SERVICE_BUS_CONNECTION_STRING` environment variable set.
    This is the connection string for your Azure Service Bus.

        import os, json
        from azure.servicebus import ServiceBusClient, ServiceBusMessage
        servicebus_client = ServiceBusClient.from_connection_string(conn_str=os.environ['SERVICE_BUS_CONNECTION_STRING'])
        with servicebus_client:
            sender = servicebus_client.get_queue_sender(queue_name="test")
            with sender:
                message = ServiceBusMessage('{"hello": "world"}')
                sender.send_messages(message)

2. As soon as you publish a message, sensor will trigger an Argo workflow. Run argo list to find the workflow.
