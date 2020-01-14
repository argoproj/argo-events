# Sensor
Sensor defines a set of event dependencies (inputs) and triggers (outputs). It listens to events from one or more
gateways and act as an event dependency manager. 
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/sensor.png?raw=true" alt="Sensor"/>
</p>

<br/>

## Event dependency
A dependency is an event the sensor is waiting to happen.

## Trigger
A Trigger is the resource executed by sensor once the event dependencies are resolved.. 

## Specification
Complete specification is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md).

## Examples
Examples are located under `examples/sensors`.
