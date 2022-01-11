
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
    start: time_range_start_utc
    stop: time_range_end_utc
```

## How it works

Time filter takes a `start` and `stop` time in `HH:MM:SS` format in UTC.

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
          start: "09:09:09"
          stop: "19:19:19"
```

If `stop` is smaller than `start` (`stop` < `start`), the stop time is treated as next day of `start`.

**Note**: `start` is inclusive while `stop` is exclusive.

The example below illustlates this behavior.

## Practical example

Here an example of time filter:

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

## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
