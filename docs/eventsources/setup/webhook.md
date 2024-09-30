# Webhook

Webhook event-source exposes a http server and allows external entities to trigger workloads via
http requests.

## Event Structure

The structure of an event dispatched by the event-source to the sensor looks like following,

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
              "header": {/* the headers from the request received by the event-source from the external entity */},
              "body": { /* the payload of the request received by the event-source from the external entity */},
            }
        }

## Specification

Webhook event-source specification is available [here](../../APIs.md#argoproj.io/v1alpha1.WebhookContext).

## Setup

1.  Install the event source in the `argo-events` namespace.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

1.  The event-source pod is listening for HTTP requests on port `12000` and endpoint `/example`.
    It's time to create the sensor.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/webhook.yaml

1.  Once the sensor pod is in running state, test the setup by sending a POST request to event-source service.

## Troubleshoot

Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
