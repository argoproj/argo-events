# Azure Event Hubs

Azure Event Hubs Trigger allows a sensor to publish events to [Azure Event Hubs](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-about). Argo Events integrates with Azure Event Hubs to stream data from an `EventSource`

**NOTE:** Parametrization for `fqdn` and `hubName` values are not yet supported.

## Specification

The Azure Event Hubs trigger specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#azureeventhubstrigger).

## Send an Event to Azure Event Hubs

1. Make sure to have the eventbus deployed in the namespace.

1. [Create an event hub](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-create)

1. Make sure that the Shared Access Key used to connect to Azure Event Hubs has the `Send` policy.

1. Get the `Primary Key` of the Shared Access Policy, the `Name` of the Shared Access Policy, the `Hub Name`, and the `FQDN` of the Azure Event Hubs Namespace

1. Create a secret called `azure-event-hubs-secret` as follows:

    **NOTE: `sharedAccessKey` refers to the `Primary Key` and `sharedAccessKeyName` refers to the Name of the Shared Access Policy.**

        apiVersion: v1
        kind: Secret
        metadata:
          name: azure-event-hubs-secret
        type: Opaque
        data:
          sharedAccessKey: <base64-shared-access-key>
          sharedAccessKeyName: <base64-shared-access-key-name>

1. Let's set up a webhook event-source to process incoming requests.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

1. Create the sensor with the following template. Replace the necessary values for `fqdn` and `hubName`:

        apiVersion: argoproj.io/v1alpha1
        kind: Sensor
        metadata:
          name: azure-events-hub
        spec:
          dependencies:
            - name: test-dep
              eventSourceName: webhook
              eventName: example
          triggers:
            - template:
              name: azure-eventhubs-trigger
              azureEventHubs:
              # FQDN of the EventsHub namespace you created
              # More info at https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string
                fqdn: eventhubs_fqdn
                sharedAccessKeyName:
                  name: azure-event-hubs-secret
                  key: sharedAccessKeyName
                sharedAccessKey:
                  name: azure-event-hubs-secret
                  key: sharedAccessKey
                # Event Hub path/name
                hubName: hub_name
                payload:
                  - src:
                      dependencyName: test-dep
                      dataKey: body.message
                    dest: message

1. The Event needs a body. In order to construct a messaged based on your event data, the Azure Event Hubs sensor has the `payload` field as part of the trigger.

    The `payload` contains the list of `src` which refers to the source events and `dest` which refers to destination key within the resulting request payload.

   The `payload` declared above will generate a message body like below,

        {
            "message": "some message here" // name/key of the object
        }

1. Let's expose the webhook event-source pod using `port-forward` so that we can make a request to it.
  
        kubectl -n argo-events port-forward <name-of-event-source-pod> 12000:12000   

1. Use either Curl or Postman to send a post request to the `http://localhost:12000/example`

        curl -d '{"message":"ok"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. Verify Events have been in ingested in Azure Events Hub by creating a [listener app](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-go-get-started-send#receive-events) or following other [code samples](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-samples). You can optionally create an [Azure Event Hubs Event Source](https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/azure-event-hubs-sensor.yaml).
