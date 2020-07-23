# Calendar

Calendar event-source generates events on either a cron schedule or an interval and helps sensor trigger workloads.

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
              "eventTime": {/* UTC time of the event */},
              "userPayload": { /* static payload available in the event source */},
            }
        }

## Specification

Calendar event-source specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/event-source.md#calendareventsource).

## Setup

1. Install the event source in the `argo-events` namespace,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/calendar.yaml

1. The event-source will generate events at every 10 seconds. Let's create the sensor,
    
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/calendar.yaml   

1. Once the sensor pod is in running state, wait for next interval to occur for sensor to trigger workflow.

## Troubleshoot
Please read the [FAQ](https://argoproj.github.io/argo-events/FAQ/).
