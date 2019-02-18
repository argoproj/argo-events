# Argo Events - The Event-Based Dependency Manager for Kubernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj/argo-events)](https://goreportcard.com/report/github.com/argoproj/argo-events)
[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/argo-events-logo.png?raw=true" alt="Sublime's custom image"/>
</p>

## What is Argo Events?
**Argo Events** is an event-based dependency manager for Kubernetes which helps you define multiple dependencies from a variety of event sources like webhook, s3, schedules, streams etc.
and trigger Kubernetes objects after successful event dependencies resolution

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/argo-events-top-level.png?raw=true" alt="Sublime's custom image"/>
</p>

<br/>

## Features 
* Manages dependencies from a variety of event sources
* Ability to customize business-level constraint logic for event dependencies resolution
* Manages everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.
* Ability to extends framework to add you own event source listener
* Supports arbitrary boolean logic to resolve event dependencies
* CloudEvents compliant

## Core Concepts
The framework is made up of two components: 

 1. **`Gateway`** which is implemented as a Kubernetes-native Custom Resource Definition that processes events from event source.

 2. **`Sensor`** which is implemented as a Kubernetes-native Custom Resource Definition that defines a set of event dependencies and triggers actions.

## Install

* ### Requirements
  * Kubernetes cluster >v1.9
  * Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.9.0

* ### Helm Chart

    Make sure you have helm client installed and Tiller server is running. To install helm, follow https://docs.helm.sh/using_helm/

    1. Add `argoproj` repository

    ```bash
    helm repo add argo https://argoproj.github.io/argo-helm
    ```

    2. Install `argo-events` chart
    
    ```bash
    helm install argo/argo-events -name argo-events
    ```   

* ### Using kubectl
  * Deploy Argo Events SA, Roles, ConfigMap, Sensor Controller and Gateway Controller
  
    ```
    kubectl create namespace argo-events
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-sa.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-cluster-roles.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-crd.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-crd.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-configmap.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-deployment.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-configmap.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-deployment.yaml
    ```



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
