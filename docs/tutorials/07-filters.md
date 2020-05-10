# Filters

In previous sections, you have seen how to trigger an Argo workflow based on events. In this tutorial,
you will learn how to apply filters on event data and context. Filters provide a powerful mechanism to
apply constraints on the events in order to determine a validity.

Argo Events offers 3 types of filters:

1. Data Filter
2. Context Filter
3. Time Filter

## Prerequisite
Webhook gateway must be set up.

## Data Filter
Data filter as the name suggests are applied on the event data. A CloudEvent from Webhook gateway has
payload structure as,


        {
            "context": {
              "type": "type_of_gateway",
              "specVersion": "cloud_events_version",
              "source": "name_of_the_gateway",
              "eventID": "unique_event_id",
              "time": "event_time",
              "dataContentType": "type_of_data",
              "subject": "name_of_the_event_within_event_source"
            },
            "data": {
              "header": {},
              "body": {},
            }
        }

Data Filter are applied on `data` within the payload. We will make a simple HTTP request
to webhook gateway with request data as `{"message":"this is my first webhook"}` and apply
data filter on `message`.

A data filter has following fields,


        data:
          - path: path_within_event_data
            type: types_of_the_data
            value:
              - list_of_possible_values      

### Comparator

The data filter offers `comparator` “>=”, “>”, “=”, “<”, or “<=”.

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

2. Send a HTTP request to gateway

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. You will notice that the sensor logs prints the event is invalid as the sensor expects for
   either `hello` or `hey` as the value of `body.message`.
 
4.  Send a valid HTTP request to gateway

        curl -d '{"message":"hello"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

5. Watch for a workflow with name `data-workflow-xxxx`.

## Context Filter
Similar to the data filter, you can apply a filter on the context of the event.

Change the subscriber in the webhook gateway to point it to `context-filter` sensor's URL.

1. Lets create a webhook sensor with context filter.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/07-filters/sensor-context-filter.yaml

2. Send a HTTP request to gateway

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. You will notice that the sensor logs prints the event is invalid as the sensor expects for
   either `custom-webhook` as the value of the `source`.

## Time Filter
Time filter is specially helpful when you need to make sure an event occurs between a 
certain time-frame. Time filter takes a `start` and `stop` time but you can also define just the
`start` time, meaning, there is no `end time` constraint or just the `stop` time, meaning, there is
no `start time` constraint. An example of time filter is available under `examples/sensors`.
