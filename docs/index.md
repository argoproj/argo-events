# Argo Events - The Event-driven Workflow Automation Framework

## What is Argo Events?

**Argo Events** is an event-driven workflow automation framework for Kubernetes
which helps you trigger K8s objects, Argo Workflows, Serverless workloads, etc.
on events from a variety of sources like webhooks, S3, schedules, messaging queues,
gcp pubsub, sns, sqs, etc.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/argo-events-top-level.png?raw=true" alt="High Level Overview"/>
</p>

<br/>

## Features

- Supports events from 20+ event sources.
- Ability to customize business-level constraint logic for workflow automation.
- Manage everything from simple, linear, real-time to complex, multi-source
  events.
- Supports Kubernetes Objects, Argo Workflow, AWS Lambda, Serverless, etc. as
  triggers.
- [CloudEvents](https://cloudevents.io/) compliant.

## Getting Started

Follow these [instructions](https://argoproj.github.io/argo-events/installation/)
to set up Argo Events.

## Documentation

- [Concepts](https://argoproj.github.io/argo-events/concepts/architecture/).
- [Argo Events in action](https://argoproj.github.io/argo-events/quick_start/).
- [Deep dive into Argo Events](https://argoproj.github.io/argo-events/tutorials/01-introduction/).

## Triggers

1. Argo Workflows
1. Standard K8s Objects
1. HTTP Requests / Serverless Workloads (OpenFaaS, Kubeless, KNative etc.)
1. AWS Lambda
1. NATS Messages
1. Kafka Messages
1. Slack Notifications
1. Azure Event Hubs Messages
1. Argo Rollouts
1. Custom Trigger / Build Your Own Trigger
1. Apache OpenWhisk
1. Log Trigger

## Event Sources

Argo Events supports 20+ event sources. The complete list of event sources is
available [here](https://argoproj.github.io/argo-events/concepts/event_source/).

## Who uses Argo Events?

Check the [list](https://github.com/argoproj/argo-events/blob/master/USERS.md)
to see who are **officially** using Argo Events. Please send a PR with your
organization name if you are using Argo Events.

## Community Blogs and Presentations

- [Automation of Everything - How To Combine Argo Events, Workflows & Pipelines, CD, and Rollouts](https://youtu.be/XNXJtxkUKeY)
- [Argo Events - Event-Based Dependency Manager for Kubernetes](https://youtu.be/sUPkGChvD54)
- [Argo Events Deep-dive](https://youtu.be/U4tCYcCK20w)
- [Automating Research Workflows at BlackRock](https://www.youtube.com/watch?v=ZK510prml8o)
- [Designing A Complete CI/CD Pipeline Using Argo Events, Workflows, and CD](https://www.slideshare.net/JulianMazzitelli/designing-a-complete-ci-cd-pipeline-using-argo-events-workflow-and-cd-products-228452500)
- TGI Kubernetes with Joe Beda:
  [CloudEvents and Argo Events](https://www.youtube.com/watch?v=LQbBgQnUs_k&list=PL7bmigfV0EqQzxcNpmcdTJ9eFRPBe-iZa&index=2&t=0s)
