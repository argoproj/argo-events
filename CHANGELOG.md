# Changelog

## v0.12
+ Added HTTP, AWS Lambda, OpenFaas triggers.
+ Support for special operations like submit, resubmit, retry, etc. for Argo workflow triggers
+ Support for Create and Update operations for K8s resource triggers.
+ Added Redis, Emitter, Stripe, NSQ, and Azure Events Hub gateways.
+ Simplified gateway subscriber logic.

## v0.12-rc
+ Support Event Source as K8s custom resource #377 

## v0.11
+ Fix transform issue when doing go build #389
+ Workflow parameters are ignored when triggered through argo-events #373
+ Allow regular expression in event filters #360
+ volumes doesn't work in Workflow triggered webhook-sensor #342

## v0.10
+ Added ability to refer the eventSource in a different namespace #311
+ Ability to send events sensors in different namespace #317
+ Support different trigger parameter operations #319
+ Support fetching/checkouts of non-branch/-tag git refs #322
+ Feature/support slack interaction actions #324
+ Gcp Pubsub Gateway Quality of life Fixes #326
+ Fix circuit bug #327
+ Pub/Sub: multi-project support #330
+ Apply resource parameters before defaulting namespace #331
+ Allow watching of resources in all namespaces at once #334
+ upport adding Github hooks without requiring a Github webhook id to be hardcoded #352

## v0.9
+ Support applying parameters for complete trigger spec #230
+ Increased test coverage #220
+ Gateway and Sensor pods are managed by the framework in the event of deletion #194
+ Add URL to webhook like gateways #216
+ Improved file gateway with path regex support #213
+ TLS support for webhook #206
+ Bug fix for Github gateway #243

## v0.8
+ Support for arbitrary boolean logic to resolve event dependencies #12
+ Ability to add timezone and extra user payload to calendar gateway #164
+ Data filter bug fix #165
+ Added multi-fields/multi-values data filter #167
+ Added support for backoff option when making connections in stream gateway #168
+ AWS SNS Gateway #169
+ GCP PubSub Gateway #176
+ Support for git as trigger source #171
+ Enhance Gitlab & Github gateway with in-build http server #172

## v0.7
+ Refactor gateways #147
+ Fixed sensor error state recovery bug #145
+ Ability to add annotations on sensor and gateway pods #143
+ Github gateway
+ Added support for NATS standard and streaming as communication protocol between gateway
  and sensor #99

## v0.6
+ Gitlab Gateway #120
+ If sensor is repeatable then deploy it as deployment else job #109
+ Start gateway containers in correct order. Gateway transformer > gateway processor. Add readiness probe to gateway transformer #106
+ Kubernetes configmaps as artifact locations for triggers #104
+ Let user set extra environment variable for sensor pod #103 
+ Ability to provide complete Pod spec in gateway.
+ The schedule for calendar gateway is now in standard cron format   #102
+ FileWatcher as core gateway #98
+ Tutorials on setting up pipelines #105
+ Asciinema recording for setup #107
+ StorageGrid Gateway
+ Scripts to generate swagger specs from gateway structs
+ Added support to pass non JSON payload to trigger
+ Update shopify's sarama to 1.19.0 to support Kafka 2.0


## v0.5
[#92](https://github.com/argoproj/argo-events/pull/92)
+ Introduced gateways as event generators. 
+ Added multiple flavors of gateway - core gateways, gRPC gateways, HTTP gateways, custom gateways
+ Added K8 events to capture gateway configurations update and as means to update gateway resource
+ SLA violations are now reported through k8 events
+ Sensors can trigger Argo workflow, any kubernetes resource and gateway
+ Gateway can send events to other gateways and sensors
+ Added examples for gateway and sensors
+ Sensors are now repeatable and fixed all issues with signal repeatability.
+ Removed signal deployments as microservices.

## v0.5-beta1 (tbd)
+ Signals as separate deployments [#49](https://github.com/argoproj/argo-events/pull/49)
+ Fixed code-gen bug [#46](https://github.com/argoproj/argo-events/issues/46)
+ Filters for signals [#26](https://github.com/argoproj/argo-events/issues/26)
+ Inline, file and url sources for trigger workflows [#41](https://github.com/argoproj/argo-events/issues/41)

## v0.5-alpha1
+ Initial release
