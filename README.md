# Argo Events - The Event-Based Dependency Manager for Kubernetes

## What is Argo Events?
Argo Events is an open source event-based dependency manager for Kubernetes. The core concept of the project are `sensors` which are implemented as Kubernetes-native Custom Resource Definition(CRD) that define a set of dependencies (signals) and actions (triggers). The sensor's triggers will only be fired after it's signals have been resolved. `Sensors` can trigger once or repeatedly.
- Define multiple dependencies from a variety of signal sources
- Define dependency constraints and build plugins to support business-level constraint logic
- Trigger messages and Kubernetes object creation after successful dependency resolution
- Trigger escalation after errors, or dependency constraint failures
- Build and manage a distributed, cross-team, event-driven architecture
- Easily leverage Kubernetes-native APIs to monitor dependencies

## Why Argo Events?
- Containers. Designed from the ground-up as Kubernetes-native. 
- Extremely lightweight. All signals, with the exception of calendar-based signals, are event-driven, meaning that there is no polling involved.
- High performance. Each `sensor` runs in its own Kubernetes job, enabling high bandwidth for processing near real-time events.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.

## Getting Started
Argo Events is a Kubernetes CRD which can manage dependencies using kubectl commands.
- [Learn about signals](./docs/signal-guide.md)
- [Learn about triggers](./docs/trigger-guide.md)
- [Review Sensor API](./docs/sensor-api.md)
- [Getting started](./docs/quickstart.md)
- [Want to contribute?](./CONTRIBUTING.md)
- See where the project is headed in the [roadmap](./ROADMAP.md)
