# Axis - The Event-Based Dependency Manager for Kubernetes
[![Build Status](https://travis-ci.org/blackrock/axis.svg?branch=master)](https://travis-ci.org/blackrock/axis) [![License Apache 2.0](https://img.shields.io/badge/License-Apache2-brightgreen.svg)](https://img.shields.io/badge/License-Apache2-brightgreen.svg)

## What is Axis?
Axis is an open source container-native event-based dependency manager for Kubernetes. The core concept of the project are `sensors` which are implemented as a Kubernetes-native Custom Resource Definition that define a set of dependencies (signals) and actions (triggers). The sensor's triggers will only be fired after all of it's signals have been satisfied. `Sensors` can be short-lived (once and done) or repeated.
- Define multiple dependencies from a variety of sources
- Define dependency constraints and build plugins to support business-level constraint logic
- Trigger messages and Kubernetes object creation after successful dependency resolution
- Trigger escalation after errors, or dependency constraint failures
- Build and manage a distributed, cross-team, event-driven architecture
- Easily leverage Kubernetes native APIs to monitor dependencies

## Why Axis?
- Containers. Axis is designed from the ground-up as Kubernetes native. 
- Extremely lightweight. All signals, with exception of calendar based signals, are event-driven, meaning there is no polling involved.
- High performance. Each Axis `sensor` runs in its own Kubernetes job enabling high bandwidth for processing near-real time events.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source batch job dependencies.

## Typical Use Casees
- Trigger processes to run after a single dependent event. 
- Manage multiple external dependencies for a complex downstream process.

## Getting Started
Axis is a Kubernetes CRD which can manage dependencies using kubectl commands.
- [Learn about signals](./docs/signal-guide.md)
- [Learn about triggers](./docs/trigger-guide.md)
- [Review Sensor API](./docs/sensor-api.md)
- [Getting started](./docs/quickstart.md)
- [Want to contribute?](./CONTRIBUTING.md)
- See where the project is headed in the [roadmap](./ROADMAP.md)


Disclaimer: This is not an officially supported BlackRock or Aladdin product
