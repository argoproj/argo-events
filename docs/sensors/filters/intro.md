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

## Logical operators

`TODO`

## Prerequisite

A webhook event-source must be set up.

## Examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
