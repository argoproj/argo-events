# Argo Events - The Event-Based Dependency Manager for Kubernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj/argo-events)](https://goreportcard.com/report/github.com/argoproj/argo-events)
[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/argo-events-logo.png?raw=true" alt="Sublime's custom image"/>
</p>

## What is Argo Events?
Argo Events is an event-based dependency manager for Kubernetes which helps you define multiple dependencies from a variety of event sources like webhook, s3, schedules, streams etc.
and trigger Kubernetes objects after successful event dependencies resolution


<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/argo-events-top-level.png?raw=true" alt="Sublime's custom image"/>
</p>

## Features
- Runtime agnostic. The first runtime and package agnostic event framework for Kubernetes.
- Containers. Designed from the ground-up as Kubernetes-native. 
- Extremely lightweight. All gateways, with the exception of calendar-based gateways, are event-driven, meaning that there is no polling involved.
- Configurable. Configure gateways at runtime
- Scalable & Resilient.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.

<br/>
<br/>

![](docs/architecture.png)

<br/>
<br/>

## Getting Started
[![asciicast](https://asciinema.org/a/207973.png)](https://asciinema.org/a/207973)

<br/>
<br/>

1. [Installation](./docs/quickstart.md)
2. [Sensor and gateway controllers](docs/controllers-guide.md)
3. [Learn about gateways](docs/gateway-guide.md)
4. [Learn about sensors](docs/sensor-guide.md)
5. [Learn about triggers](docs/trigger-guide.md)
6. Install Gateways and Sensors
    1. [Webhook](gateways/core/webhook/install.md)
    2. [Artifact](gateways/core/artifact/install.md)
    3. [Calendar](gateways/core/calendar/install.md)
    4. [Resource](gateways/core/resource/install.md)
    5. [File](gateways/core/file/install.md)
    6. Streams
        1. [NATS](gateways/core/stream/nats/install.md)
        2. [KAFKA](gateways/core/stream/kafka/install.md)
        3. [AMQP](gateways/core/stream/amqp/install.md)
        4. [MQTT](gateways/core/stream/mqtt/install.md)
7. [Write your own gateway](docs/custom-gateway.md)
8. [Want to contribute or develop/run locally?](./CONTRIBUTING.md)
9. See where the project is headed in the [roadmap](./ROADMAP.md)
