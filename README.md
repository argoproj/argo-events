# Argo Events - The Event-Based Dependency Manager for Kubernetes

## What is Argo Events?
Argo Events is an open source event-based dependency manager for Kubernetes. The core concept of the project are `sensors` which are implemented as Kubernetes-native Custom Resource Definition that define a set of dependencies (signals) and actions (triggers). The sensor's triggers will only be fired after it's signals have been resolved. `Sensors` can trigger once or repeatedly.
- Define multiple dependencies from a variety of signal sources
- Build custom signal microservices to support business-level constraint logic
- Trigger messages and Kubernetes object creation after successful dependency resolution
- Trigger escalation after errors, or dependency constraint failures
- Build and manage a distributed, cross-team, event-driven architecture
- Easily leverage Kubernetes-native APIs to monitor dependencies

## Why Argo Events?
- Runtime agnostic. The first runtime and package agnostic event framework for Kubernetes.
- Containers. Designed from the ground-up as Kubernetes-native. 
- Extremely lightweight. All signals, with the exception of calendar-based signals, are event-driven, meaning that there is no polling involved.
- Configurable. Choose which signals to support and only deploy those you want to Kubernetes.
- Scalability & Resilient. Signal sources run as stateless microservices enabling you to easily scale.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.

## Getting Started
Argo Events is a Kubernetes CRD which can manage dependencies using kubectl commands.
- [Learn about signals](./docs/signal-guide.md)
- [Learn about triggers](./docs/trigger-guide.md)
- [Getting started](./docs/quickstart.md)
- [Want to contribute or develop/run locally?](./CONTRIBUTING.md)
- See where the project is headed in the [roadmap](./ROADMAP.md)
