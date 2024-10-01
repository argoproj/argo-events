# Azure Service Bus

Service Bus Trigger allows a sensor to send messages to Azure Service Bus queues and topics.

## Specification

The Azure Service Bus trigger specification is available [here](../../APIs.md#argoproj.io/v1alpha1.AzureServiceBusTrigger).

## Setup

1.  Create a queue called `test` either using Azure CLI or Azure Service Bus management console.

1.  Fetch your connection string for Azure Service Bus and base64 encode it.

1.  Create a secret called `azure-secret` as follows.

        apiVersion: v1
        kind: Secret
        metadata:
          name: azure-secret
        type: Opaque
        data:
          connectionstring: <base64-connection-string>

1.  Deploy the secret.

        kubectl -n argo-events apply -f azure-secret.yaml

1.  Let's set up a webhook event-source to process incoming requests.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

1.  Create a sensor by running the following command.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/azure-service-bus-sensor.yaml

1.  The Service Bus message needs a body. In order to construct a messaged based on your event data, the Azure Service Bus sensor has the payload field as part of the trigger.

    The payload declared above will generate a message body like below,

        {
            "message": "some message here" // name/key of the object
        }

1.  Let's expose the webhook event-source pod using port-forward so that we can make a request to it.

        kubectl -n argo-events port-forward <name-of-event-source-pod> 12000:12000

1.  Use either Curl or Postman to send a post request to the http://localhost:12000/example.

        curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
