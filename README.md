# Argo Events - The Event-Based Dependency Manager for Kubernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj/argo-events)](https://goreportcard.com/report/github.com/argoproj/argo-events)

## What is Argo Events?
Argo Events is an open source event-based dependency manager for Kubernetes. The core concept of the project are
 * `gateways` which are implemented as a Kubernetes-native Custom Resource Definition that either produce the events internally or process the events that originate from outside the gateways
    
 * `sensors` which are implemented as a Kubernetes-native Custom Resource Definition that define a set of dependencies and trigger actions.

    - Define multiple dependencies from a variety of gateway sources
    - Build custom gateways to support business-level constraint logic
    - Trigger messages and Kubernetes object creation after successful dependency resolution
    - Trigger escalation after errors, or dependency constraint failures
    - Build and manage a distributed, cross-team, event-driven architecture
    - Easily leverage Kubernetes-native APIs to monitor dependencies

## Why Argo Events?
- Runtime agnostic. The first runtime and package agnostic event framework for Kubernetes.
- Containers. Designed from the ground-up as Kubernetes-native. 
- Extremely lightweight. All gateways, with the exception of calendar-based gateways, are event-driven, meaning that there is no polling involved.
- Configurable. Configure gateways at runtime
- Scalable & Resilient.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.

## Getting Started
- [Learn about gateways](./docs/gateway-guide.md)
- [Learn about sensors](./docs/sensor-guide.md)
- [Learn about triggers](./docs/trigger-guide.md)
- [Getting started](./docs/quickstart.md)
- [User Guide](./docs/tutorial.md)
- [Want to contribute or develop/run locally?](./CONTRIBUTING.md)
- See where the project is headed in the [roadmap](./ROADMAP.md)
