
# Expr filter

Expr filters are applied to the event data. A CloudEvent from Webhook event-source has payload structure as:

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

Expr filters are applied on `data` within the payload.

## Fields

An expr filter has following fields:

```yaml
filters:
  exprLogicalOperator: logical_operator_applied
  expr:
    - expr: expression_to_evalute
      fields:
        - name: parameter_name
          path: path_to_parameter_value
```

> ⚠️ `PLEASE NOTE` order in which expr filters are declared corresponds to the order in which the Sensor will evaluate them.

## Logical operator

Expr filters can be evaluated together in 2 ways:

- `and`, meaning that all expr filters returning `true` are required for an event to be valid
- `or`, meaning that only one expr filter returning `true` is enough for an event to be valid

Any kind of error is considered as `false` (e.g. path not existing in event body).

Such behaviour can be configured with `exprLogicalOperator` field in a Sensor dependency filters, e.g.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: data-filters-example
spec:
  dependencies:
    - name: sample-dependency
      eventSourceName: webhook
      eventName: sample-event
      filters:
        exprLogicalOperator: "or"
        exprs:
          - expr: a == "b" || c != 10
            fields:
              - name: a
                path: a
              - name: c
                path: c
          - expr: e == false
            fields:
              - name: e
                path: d.e
          # ...
```

Available values:

- `""` (empty), defaulting to `and`
- `and`, default behaviour
- `or`

> ⚠️ `PLEASE NOTE` Expr logical operator values must be `lower case`.

## How it works

The `expr` field defines the expression to be evaluated. The `fields` stanza defines `name` and `path` of each parameter used in the expression.

`name` is arbitrary and used in the `expr`, `path` defines how to find the value in the data payload then to be assigned to a parameter.

The expr filter evaluates the expression contained in `expr` using [govaluate](https://github.com/Knetic/govaluate). This library leverages an incredible flexibility and power.

With govaluate we are able to define complex combination of arithmetic (`-`, `*`, `/`, `**`, `%`), negation (`-`), inversion (`!`), bitwise not (`~`), logical (`&&`, `||`), ternary conditional (`?`, `:`) operators, 
together with comparators (`>`, `<`, `>=`, `<=`), comma-separated arrays and custom functions.

Here some examples:

- `action =~ "start"`
- `action == "end" && started == true`
- `action =~ "start" || (started == true && instances == 2)`

To discover all options offered by govaluate, take a look at its [manual](https://github.com/Knetic/govaluate/blob/master/MANUAL.md).

## Partical example

1. Create a webhook event-source

  ```bash
  kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/event-sources/webhook.yaml
  ```

1. Create a webhook sensor with expr filter

  ```bash
  kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/filter-with-expressions.yaml
  ```

1. Send an HTTP request to event-source

  ```bash
  curl -d '{ "a": "b", "c": 11, "d": { "e": true } }' -H "Content-Type: application/json" -X POST http://localhost:12000/example
  ```

1. You will notice in sensor logs that the event is invalid as the sensor expects `e == false`

1. Send another HTTP request to event-source

  ```bash
  curl -d '{ "a": "b", "c": 11, "d": { "e": false } }' -H "Content-Type: application/json" -X POST http://localhost:12000/example
  ```

1. Look for a workflow with name starting with `expr-workflow-`


## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
