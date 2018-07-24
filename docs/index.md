# Argo Events Documentation

## Why Argo Events?
- Containers. Designed from the ground-up as Kubernetes-native.
- Extremely lightweight. All signals, with exception of calendar based signals, are event-driven, meaning there is no polling involved.
- Configurable. Choose which signals to support and only deploy those you want to Kubernetes.
- Scalability & Resilient. Signal sources run as stateless microservices enabling you to scale simply by increasing the deployment replicas.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source batch job dependencies.

## Basics
Argo Events is an open source event-based dependency manager for Kubernetes. The core concept of the project are `sensors` which are implemented as a Kubernetes-native Custom Resource Definition that define a set of dependencies (inputs) and actions (outputs). The sensor's actions will only be triggered after it's dependencies have been resolved.
- Define multiple dependencies from a variety of sources
- Plugin custom signal microservices to support business-level constraint logic
- Trigger messages and Kubernetes object creation after successful dependency resolution
- Trigger escalation after errors, or dependency constraint failures
- Build and manage a distributed, cross-team, event-driven architecture
- Easily leverage Kubernetes native APIs to monitor dependencies

## Learn More
- [Signals](signal-guide.md)
- [Triggers](trigger-guide.md)
