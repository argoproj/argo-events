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
* Manage dependencies from 20+ event sources.
* Ability to customize business-level constraint logic for event dependencies resolution.
* Manage everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.
* Supports AWS Lambda and OpenFaas as triggers.
* Supports integration of existing API servers with 20+ event sources.
* CloudEvents compliant.

## Event Sources
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
18. Stripe
19. NSQ
20. Emitter
21. Redis
22. Azure Events Hub


## Triggers

1. Argo Workflows
1. Standard K8s Objects
1. HTTP Requests / Serverless (OpenFaas, Kubeless, KNative etc.)
1. AWS Lambda
1. NATS Messages
1. Kafka Messages
1. Slack Notifications
1. Argo Rollouts CR
1. Custom Triggers
1. Apache OpenWhisk [Coming Soon...]


## Installation
Follow the [setup](https://argoproj.github.io/argo-events/installation/) to install Argo Events.

## Quick Start
Check out the quick start [guide](https://argoproj.github.io/argo-events/quick_start/) to trigger Argo workflows on webhook events.

## Deep Dive
Explore the [tutorials](https://argoproj.github.io/argo-events/tutorials/01-introduction/) to dive deep into Argo Events features.
