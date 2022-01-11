
# Context Filter

Context filter is applied to the event context. A CloudEvent from Webhook event-source has payload structure as:

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
      "body": {},
    }
}
```

## Fields

Context filter has following fields:

```yaml
filters:
  context:
    type: event_type
    subject: event_type
    source: event_source
    datacontenttype: event_type
```

You can also specify id, specversion and time fields in the YAML manifest, but they are ignored.

## How it works

Context filter takes a `start` and `stop` time in `HH:MM:SS` format in UTC.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: with-ctx-filter
spec:
  template:
    serviceAccountName: operate-workflow-sa
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
      filters:
        context:
          source: webhook
```

## Practical example

Change the subscriber in the webhook event-source to point it to `context-filter` sensor's URL.

1. Lets create a webhook sensor with context filter.

  ```bash
  kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/filter-with-context.yaml
  ```

1. Send a HTTP request to event-source.

  ```bash
  curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
  ```

1. You will notice that the sensor logs prints the event is invalid as the sensor expects for
   either `custom-webhook` as the value of the `source`.

## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
