# Argo Events - The Event-Based Dependency Manager for Kubernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj/argo-events)](https://goreportcard.com/report/github.com/argoproj/argo-events)
[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)
[![Build Status](https://travis-ci.org/argoproj/argo-events.svg?branch=master)](https://travis-ci.org/argoproj/argo-events)
[![Coverage Status](https://coveralls.io/repos/github/argoproj/argo-events/badge.svg)](https://coveralls.io/github/argoproj/argo-events)
[![GoDoc](https://godoc.org/github.com/argoproj/argo-events?status.svg)](https://godoc.org/github.com/argoproj/argo-events/pkg/apis)	
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

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
* Supports CloudEvents for describing event data and transmission.
* Ability to manage event sources at runtime.

## Documentation
To learn more about Argo Events, [go to complete documentation](https://argoproj.github.io/argo-events/)

## Contribute
Read and abide by the [Argo Events Code of Conduct](https://github.com/argoproj/argo-events/blob/master/CODE_OF_CONDUCT.md)

[Contributions](https://github.com/argoproj/argo-events/issues) are more than welcome, if you are interested please take a look at our [Contributing Guidelines](./CONTRIBUTING.md).

## License
Apache License Version 2.0, see [LICENSE](./LICENSE)
