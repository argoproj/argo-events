# Sensor
Sensors define a set of event dependencies (inputs) and triggers (outputs). 
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/sensor.png?raw=true" alt="Sensor"/>
</p>

<br/>

## What is an event dependency?
A dependency is an event the sensor is expecting to happen. It is defined as "gateway-name:event-source-name".
Also, you can use [globs](https://github.com/gobwas/glob#syntax) to catch a set of events (e.g. "gateway-name:*").

## Specification
Complete specification is available [here]()
