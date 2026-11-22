# WebSocket

The event-source connects to a WebSocket server and dispatches every incoming
message to the eventbus, helping the sensor trigger workloads. The connection
is re-established automatically with the configured backoff when it drops.

## Event Structure

The structure of an event dispatched by the event-source over the eventbus looks like following,

        {
            "context": {
               "type": "websocket",
               "specversion": "cloud_events_version",
               "source": "name_of_the_event_source",
               "id": "unique_event_id",
               "time": "event_time",
               "datacontenttype": "type_of_data",
               "subject": "name_of_the_configuration_within_event_source"
            },
            "data": {
                "body": "Body is the message payload",
                "metadata": "user defined metadata passed along the event payload"
            }
        }

## Specification

WebSocket event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.WebSocketEventSource).

## Setup

1.  Make sure the WebSocket server is reachable from the event-source pod.

1.  Create the event source by running the following command. Make sure to update the appropriate fields.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/websocket.yaml

1.  Create a sensor that listens to the `websocket` event source, for example with a `websocket` dependency in the sensor triggers.
