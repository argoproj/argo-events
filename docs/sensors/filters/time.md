
# Time Filter

Time filter is applied to the event time, contained in the event context. A CloudEvent from Webhook event-source has payload structure as:

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

It filters out events occurring outside the specified time range, so it is specially helpful when
you need to make sure an event occurs between a certain time-frame.

## Fields

Time filter has following fields:

```yaml
filters:
  time:
    start: time_range_start
    stop: time_range_end
    timezone: timezone_name  # Optional, defaults to UTC
```

## How it works

Time filter takes a `start` and `stop` time in `HH:MM:SS` format. By default, times are interpreted in UTC. You can optionally specify a `timezone` field using IANA timezone names (e.g., `America/New_York`, `Europe/London`, `Asia/Tokyo`) to handle different timezones and automatically adapt to daylight saving time changes.

### Example with UTC (default)

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: with-time-filter
spec:
  template:
    serviceAccountName: operate-workflow-sa
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
      filters:
        time:
          start: "02:30:00"
          stop: "04:30:00"
```

### Example with timezone

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: with-time-filter-tz
spec:
  template:
    serviceAccountName: operate-workflow-sa
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
      filters:
        time:
          start: "09:00:00"
          stop: "17:00:00"
          timezone: "America/New_York"  # 9 AM to 5 PM Eastern Time
```

If `stop` is smaller than `start` (`stop` < `start`), the stop time is treated as next day of `start`.

**Note**: `start` is inclusive while `stop` is exclusive.

### Time filter behaviour visually explained

1. if `start` < `stop`: event time must be in `[start, stop)`.

     00:00:00                            00:00:00                            00:00:00
     ┃     start                   stop  ┃     start                   stop  ┃
    ─┸─────●───────────────────────○─────┸─────●───────────────────────○─────┸─
           ╰───────── OK ──────────╯           ╰───────── OK ──────────╯

1. if `stop` < `start`: event time must be in `[start, stop@Next day)`  
   (this is equivalent to: event time must be in `[00:00:00, stop) || [start, 00:00:00@Next day)`).

     00:00:00                            00:00:00                            00:00:00
     ┃           stop        start       ┃       stop            start       ┃
    ─┸───────────○───────────●───────────┸───────────○───────────●───────────┸─
    ─── OK ──────╯           ╰───────── OK ──────────╯           ╰────── OK ───

## Practical example

1. Create a webhook event-source

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml

1. Create a webhook sensor with time filter

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/filter-with-time.yaml

1. Send an HTTP request to event-source

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

1. You will notice one of following behaviours:

    - if you run this example between 02:30 and 04:30, the sensor logs the event is valid
    - if you run this example outside time range between 02:30 and 04:30, the sensor logs the event is invalid

## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
