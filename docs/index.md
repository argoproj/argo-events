# Argo Events

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

