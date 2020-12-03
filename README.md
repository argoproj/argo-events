# Argo Events - The Event-driven Workflow Automation Framework - DERP

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj/argo-events)](https://goreportcard.com/report/github.com/argoproj/argo-events)
[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)
[![Build Status](https://travis-ci.org/argoproj/argo-events.svg?branch=master)](https://travis-ci.org/argoproj/argo-events)
[![GoDoc](https://godoc.org/github.com/argoproj/argo-events?status.svg)](https://godoc.org/github.com/argoproj/argo-events/pkg/apis)	
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## What is Argo Events?

**Argo Events** is an event-driven workflow automation framework for Kubernetes. It allows you to trigger 10 different actions (such as the creation of Kubernetes objects, invoke workflows or serverless workloads) on over 20 different events (such as webhook, S3 drop, cron schedule, messaging queues - e.g. Kafaka, GCP PubPub, SNS, SQS).

[![Argo Events in 3 minutes](https://img.youtube.com/vi/Aqi1zyTpM44/0.jpg)](https://youtu.be/Aqi1zyTpM44)

## Features 

* Supports events from [20+ event sources](https://argoproj.github.io/argo-events/concepts/event_source/) and [10+ triggers](https://argoproj.github.io/argo-events/concepts/trigger/).
* Ability to customize business-level constraint logic for workflow automation.
* Manage everything from simple, linear, real-time to complex, multi-source events.
* [CloudEvents](https://cloudevents.io/) compliant.

## Getting Started

Follow these [instruction](https://argoproj.github.io/argo-events/installation/) to set up Argo Events.

## Documentation

- [Concepts](https://argoproj.github.io/argo-events/concepts/architecture/)
- [Argo Events in action](https://argoproj.github.io/argo-events/quick_start/)
- [Deploy event-sources and sensors](https://argoproj.github.io/argo-events/setup/webhook/)
- [Deep dive into Argo Events](https://argoproj.github.io/argo-events/tutorials/01-introduction/)

## Blogs and Presentations

* [Argo Events Deep-dive](https://youtu.be/U4tCYcCK20w)
* [Automating Research Workflows at BlackRock](https://www.youtube.com/watch?v=ZK510prml8o)
* [Designing A Complete CI/CD Pipeline CI/CD Pipeline Using Argo Events, Workflows, and CD](https://www.slideshare.net/JulianMazzitelli/designing-a-complete-ci-cd-pipeline-using-argo-events-workflow-and-cd-products-228452500)
* TGI Kubernetes with Joe Beda: [CloudEvents and Argo Events](https://www.youtube.com/watch?v=LQbBgQnUs_k&list=PL7bmigfV0EqQzxcNpmcdTJ9eFRPBe-iZa&index=2&t=0s)

## Who uses Argo Events?

[Official Argo Events user list](USERS.md)

## Contribute

Read and abide by the [Argo Events Code of Conduct](https://github.com/argoproj/argo-events/blob/master/CODE_OF_CONDUCT.md).

[Contributions](https://github.com/argoproj/argo-events/issues) are more than welcome, if you are interested take a look at our [Contributing Guidelines](./CONTRIBUTING.md).

## License

Apache License Version 2.0, see [LICENSE](./LICENSE)
