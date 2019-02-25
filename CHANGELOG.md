# Changelog

## v0.8
+ Support for arbitrary boolean logic to resolve event dependencies #12
+ Ability to add timezone and extra user payload to calendar gateway #164
+ Data filter bug fix #165
+ Added multi-fields/multi-values data filter #167
+ Added support for backoff option when making connections in stream gateway #168
+ AWS SNS Gateway #169
+ GCP PubSub Gateway #176
+ Support for git as trigger source #171
+ Enhance Gitlan & Github gateway with in-build http server #172


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
