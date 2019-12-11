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

## Event Listeners
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
