# Sensor

Sensor defines a set of event dependencies (inputs) and triggers (outputs).
It listens to events on the eventbus and acts as an event dependency manager to resolve and execute the triggers.

## Event dependency

A dependency is an event the sensor is waiting for.

## Specification

The complete specification is available [here](../APIs.md#argoproj.io/v1alpha1.Sensor).

## Examples

Examples are located under [examples/sensors](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
