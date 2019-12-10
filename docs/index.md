# Argo Events

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/logo.png?raw=true" alt="Logo"/>
</p>

## What is Argo Events?
**Argo Events** is an event-based dependency manager for Kubernetes which helps you define multiple dependencies from a variety of event sources like webhook, s3, schedules, streams etc.
and trigger Kubernetes objects after successful event dependencies resolution.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/argo-events-top-level.png?raw=true" alt="High Level Overview"/>
</p>

<br/>

## Features 
* Manage dependencies from a variety of event sources.
* Ability to customize business-level constraint logic for event dependencies resolution.
* Manage everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.
* Ability to extends framework to add your own event source listener.
* Define arbitrary boolean logic to resolve event dependencies.
* CloudEvents compliant.
* Ability to manage event sources at runtime.

## Core Concepts
The framework is made up of three components: 

 1. [**Gateway**](gateway.md) which is implemented as a Kubernetes-native Custom Resource Definition processes events from event source.

 2. [**Sensor**](sensor.md) which is implemented as a Kubernetes-native Custom Resource Definition defines a set of event dependencies and triggers K8s resources.

 3. **Event Source** is a configmap that contains configurations which is interpreted by gateway as source for events producing entity. 

## In Nutshell
Gateway monitors event sources and starts routines in parallel that consume events from entities like S3, Github, SNS, SQS,
PubSub etc. and dispatch these events to sensor. Sensor upon receiving the events, evaluates the dependencies and triggers Argo workflows or other K8s resources.
