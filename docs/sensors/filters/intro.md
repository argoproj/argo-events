
# Introducation

Filters provide a powerful mechanism to apply constraints on the events in order to determine a validity.

If filters determine an event is valid, this will trigger the action defined by the Sensor.

If filters determine an event is not valid, this won't trigger any action.

## Types

Argo Events offers 4 types of filters:

1. [`Expr` Filter](https://argoproj.github.io/argo-events/filters/expr)
1. [`Data` Filter](https://argoproj.github.io/argo-events/filters/data)
1. [`Context` Filter](https://argoproj.github.io/argo-events/filters/ctx)
1. [`Time` Filter](https://argoproj.github.io/argo-events/filters/time)

> ⚠️ `PLEASE NOTE` this is the order in which Sensor evaluates filter types: expr, data, context, time.

## Logical operator

Filters types can be evaluated together in 2 ways:

- `and`, meaning that all filters returning `true` are required for an event to be valid
- `or`, meaning that only one filter returning `true` is enough for an event to be valid

Any kind of filter error is considered as `false` (e.g. path not existing in event body).

Such behaviour can be configured with `filtersLogicalOperator` field in a Sensor dependency, e.g.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: multiple-filters-example
spec:
  dependencies:
    - name: sample-dependency
      eventSourceName: webhook
      eventName: sample-event
      filtersLogicalOperator: "or"
      filters:
        # ...
```

Available values:

- `""` (empty), defaulting to `and`
- `and`, default behaviour
- `or`

> ⚠️ `PLEASE NOTE` Logical operator values must be `lower case`.

## Examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
