# Argo Events Documentation

## Why Argo Events?
- Containers. Designed from the ground-up as Kubernetes-native.
- Extremely lightweight. All gateways, with exception of calendar based gateway, are event-driven, meaning there is no polling involved.
- Configurable. Configure gateways at runtime
- Extensible. Write custom gateways for any business use case in the language of your choice
- Scalability & Resilient.
- Simple or Complex dependencies. Manage everything from simple, linear, real-time dependencies to complex, multi-source batch job dependencies.

## Basics
Argo Events is an open source event-based dependency manager for Kubernetes. The core concept of the project are
 * `gateways` which are implemented as a Kubernetes-native Custom Resource Definition consumes events from event sources.
    
 * `sensors` which are implemented as a Kubernetes-native Custom Resource Definition define a set of dependencies 
 (inputs) and actions (outputs).
  

## Features  
* Define gateway to support business-level logic for producing events.
* Define multiple dependencies from a variety of sources
* Trigger Kubernetes object creation after successful dependency resolution
* Trigger escalation after errors, or dependency constraint failures
* Build and manage a distributed, cross-team, event-driven architecture
* Easily leverage Kubernetes native APIs to monitor dependencies

## Learn More
- [Quickstart](quickstart.md)
- [Gateways](gateway-guide.md)
- [Sensors](sensor-guide.md)
- [Triggers](trigger-guide.md)
