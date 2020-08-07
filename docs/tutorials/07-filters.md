# Filters

In the previous sections, you have seen how to trigger an Argo workflow based on events. In this tutorial,
you will learn how to apply filters on event data and context. Filters provide a powerful mechanism to
apply constraints on the events in order to determine a validity.

Argo Events offers 3 types of filters:

1. Data Filter
2. Context Filter
3. Time Filter

## Prerequisite

Webhook event-source must be set up.

## Data Filter
Data filter as the name suggests are applied on the event data. A CloudEvent from Webhook event-source has
payload structure as,


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

Data Filter are applied on `data` within the payload. We will make a simple HTTP request
to webhook event-source with request data as `{"message":"this is my first webhook"}` and apply
data filter on `message`.

A data filter has following fields,


        data:
          - path: path_within_event_data
            type: types_of_the_data
            value:
              - list_of_possible_values      

### Comparator

The data filter offers `comparator` “>=”, “>”, “=”, “!=”, “<”, or “<=”.

e.g.,

              filters:
                name: data-filter
                data:
                  - path: body.value
                    type: number
                    comparator: ">"
                    value:
                      - "50.0"

<br/>

**Note**: If data type is a `string`, then you can pass either an exact value or a regex.
If data types is bool or float, then you need to pass the exact value.

1. Lets create a webhook sensor with data filter.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/07-filters/sensor-data-filters.yaml

2. Send a HTTP request to event-source

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. You will notice that the sensor logs prints the event is invalid as the sensor expects for
   either `hello` or `hey` as the value of `body.message`.
 
4.  Send a valid HTTP request to event-source

        curl -d '{"message":"hello"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

5. Watch for a workflow with name `data-workflow-xxxx`.

## Context Filter
Similar to the data filter, you can apply a filter on the context of the event.

Change the subscriber in the webhook event-source to point it to `context-filter` sensor's URL.

1. Lets create a webhook sensor with context filter.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/07-filters/sensor-context-filter.yaml

2. Send a HTTP request to event-source

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. You will notice that the sensor logs prints the event is invalid as the sensor expects for
   either `custom-webhook` as the value of the `source`.

## Time Filter

You can also use time filter, which is applied on event time.
It filters out events that occur outside the specified time range, so it is specially helpful when
you need to make sure an event occurs between a certain time-frame.

Time filter takes a `start` and `stop` time in `HH:MM:SS` format in UTC. If `stop` is smaller than `start`,
the stop time is treated as next day of `start`. Note that `start` is inclusive while `stop` is exclusive.
The diagrams below illustlate these behavior.

An example of time filter is available under `examples/sensors`.

1. if `start` < `stop`: event time must be in `[start, stop)`

         00:00:00                            00:00:00                            00:00:00
         ┃     start                   stop  ┃     start                   stop  ┃
        ─┸─────●───────────────────────○─────┸─────●───────────────────────○─────┸─
               ╰───────── OK ──────────╯           ╰───────── OK ──────────╯

2. if `stop` < `start`: event time must be in `[start, stop@Next day)`  
   (this is equivalent to: event time must be in `[00:00:00, stop) || [start, 00:00:00@Next day)`)

         00:00:00                            00:00:00                            00:00:00
         ┃           stop        start       ┃       stop            start       ┃
        ─┸───────────○───────────●───────────┸───────────○───────────●───────────┸─
        ─── OK ──────╯           ╰───────── OK ──────────╯           ╰────── OK ───
