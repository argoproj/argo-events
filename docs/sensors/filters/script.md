# Script filter

Script filters can be used to filter the events with [LUA](https://www.lua.org/) scripts.

Script filters are applied to the event `data`. A CloudEvent from Webhook event-source has payload structure as:

```json
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
    "header": {},
    "body": {}
  }
}
```

## Fields

An Script filter can be defined under `filters` with a field `script`:

```yaml
filters:
  script: |-
    if event.body.a == "b" and event.body.d.e == "z" then return true else return false end
```

## Practical example

1. Create a webhook event-source

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

1. Create a webhook sensor with context filter

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/filter-script.yaml

1. Send an HTTP request to the event-source

        kubectl port-forward svc/webhook-eventsource-svc 12000
        curl -d '{"hello": "world"}' -X POST http://localhost:12000/example

1. You will notice in sensor logs that the event did not trigger anything.

1. Send another HTTP request the event-source

        curl -X POST -d '{"a": "b", "d": {"e": "z"}}' http://localhost:12000/example

1. Then you will see the event successfully triggered a workflow creation.
