# Argo Events - The Event-Based Dependency Manager for Kubernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj/argo-events)](https://goreportcard.com/report/github.com/argoproj/argo-events)
[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)
[![Build Status](https://travis-ci.org/argoproj/argo-events.svg?branch=master)](https://travis-ci.org/argoproj/argo-events)
[![GoDoc](https://godoc.org/github.com/argoproj/argo-events?status.svg)](https://godoc.org/github.com/argoproj/argo-events/pkg/apis)	
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

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
* Supports [CloudEvents](https://cloudevents.io/) for describing event data and transmission.
* Ability to manage event sources at runtime.

## Getting Started
Follow [setup](https://argoproj.github.io/argo-events/installation/) instructions for installation. To see the Argo-Events in action, follow the
[getting started](https://argoproj.github.io/argo-events/getting_started/) guide. 
Complete documentation is available at https://argoproj.github.io/argo-events/

[![asciicast](https://asciinema.org/a/AKkYwzEakSUsLqH8mMORA4kza.png)](https://asciinema.org/a/AKkYwzEakSUsLqH8mMORA4kza)

## Available Event Listeners
1. AMQP
2. AWS SNS
3. AWS SQS
4. Cron Schedules
5. GCP PubSub
6. GitHub
7. GitLab
8. HDFS
9. File Based Events
10. Kafka
11. Minio
12. NATS
13. MQTT
14. K8s Resources
15. Slack
16. NetApp StorageGrid
17. Webhooks

## Who uses Argo Events?
Organizations below are **officially** using Argo Events. Please send a PR with your organization name if you are using Argo Events.
1. [BioBox Analytics](https://biobox.io)
1. [BlackRock](https://www.blackrock.com/)
1. [Canva](https://www.canva.com/)
1. [Fairwinds](https://fairwinds.com/)
1. [InsideBoard](https://www.insideboard.com)
1. [Intuit](https://www.intuit.com/)
1. [Viaduct](https://www.viaduct.ai/)

## Community Blogs and Presentations
* [Automating Research Workflows at BlackRock](https://www.youtube.com/watch?v=ZK510prml8o)

## Contribute
Read and abide by the [Argo Events Code of Conduct](https://github.com/argoproj/argo-events/blob/master/CODE_OF_CONDUCT.md)

[Contributions](https://github.com/argoproj/argo-events/issues) are more than welcome, if you are interested please take a look at our [Contributing Guidelines](./CONTRIBUTING.md).

## License
Apache License Version 2.0, see [LICENSE](./LICENSE)
