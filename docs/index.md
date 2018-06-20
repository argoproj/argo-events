# Argo Events Documentation

## Why Argo Events?
- Containers. Designed from the ground-up as Kubernetes-native.
- Extremely lightweight. All signals, with exception of calendar based signals, are event-driven, meaning there is no polling involved.
- High performance. Each `sensor` runs in its own Kubernetes job enabling high bandwidth for processing near-real time events.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source batch job dependencies.

## Basics
Argo Events is an open source event-based dependency manager for Kubernetes. The core concept of the project are `sensors` which are implemented as a Kubernetes-native Custom Resource Definition that define a set of dependencies (inputs) and actions (outputs). The sensor's actions will only be triggered after it's dependencies have been resolved.
- Define multiple dependencies from a variety of sources
- Define dependency constraints and build plugins to support business-level constraint logic
- Trigger messages and Kubernetes object creation after successful dependency resolution
- Trigger escalation after errors, or dependency constraint failures
- Build and manage a distributed, cross-team, event-driven architecture
- Easily leverage Kubernetes native APIs to monitor dependencies

## Learn More
- [Signals](signal-guide.md)
- [Triggers](trigger-guide.md)
- [Sensor API](sensor-api.md)
