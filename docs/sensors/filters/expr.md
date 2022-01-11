
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

A data filter has following fields:

```yaml
filters:
  exprLogicalOperator: logical_operator_applied
  expr:
    - expr: expression_to_evalute
      fields:
        - name: parameter_name
        - path: path_to_parameter_value
```

> ⚠️ `PLEASE NOTE` order in which expr filters are declared corresponds to the order in which the Sensor will evaluate them.

## Logical operator

Expr filters can be evaluated together in 2 ways:

- `AND`, meaning that all expr filters returning `true` are required for an event to be valid
- `OR`, meaning that only one expr filter returning `true` is enough for an event to be valid

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

- `empty`, defaulting to `and`
- `and`, default behaviour
- `or`

## How it works

The `expr` field is evaluated using [govaluate](https://github.com/Knetic/govaluate).

`TODO`

## ??? example

`TODO`

## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
