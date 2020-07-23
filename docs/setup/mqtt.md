# MQTT

The event-source listens to messages over MQTT and helps sensor trigger the workloads.  

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,


        {
            "context": {
              "type": "type_of_event_source",
              "specVersion": "cloud_events_version",
              "source": "name_of_the_event_source",
              "eventID": "unique_event_id",
              "time": "event_time",
              "dataContentType": "type_of_data",
              "subject": "name_of_the_configuration_within_event_source"
            },
            "data": {
                "topic": "Topic refers to the MQTT topic name",
                "messageId": "MessageId is the unique ID for the message",
                "body": "Body is the message payload"
            }
        }

## Specification

MQTT event-source specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/event-source.md#mqtteventsource).

## Setup

1. Make sure to set up the MQTT Broker and Bridge in Kubernetes if you don't already have one. 

1. Create the event source by running the following command. Make sure to update the appropriate fields.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/mqtt.yaml

1. Create the sensor by running the following command,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/mqtt.yaml

1. Send message by using MQTT client.

1. Once a message is published, an argo workflow will be triggered. Run `argo list` to find the workflow. 

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).


