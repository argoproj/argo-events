# Changelog

## v0.6
[#100](https://github.com/argoproj/argo-events/pull/100)
+ Custom gateways for Slack, Twitter, AWS S3, Google Cloud Pub-Sub, Reddit and Docker #96
+ FileWatcher as core gateway #98
+ Kubernetes configmaps as artifact locations for triggers #104
+ Tutorials on setting up pipelines #105
+ Start gateway containers in correct order. Gateway transformer > gateway processor. Add readiness probe to gateway transformer #106
+ Let user set extra environment variable for sensor pod #103
+ Asciinema recording for setup #107
+ Retry strategy for sensor and gateway 
+ The schedule for calendar gateway is now in standard cron format   #102 
+ Allow multiple servers in webhook gateway #108
+ Update shopify's sarama to 1.19.0 to support Kafka 2.0
+ If sensor is repeatable then deploy it as deployment else job #109

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

## v0.5-beta1
+ Signals as separate deployments [#49](https://github.com/argoproj/argo-events/pull/49)
+ Fixed code-gen bug [#46](https://github.com/argoproj/argo-events/issues/46)
+ Filters for signals [#26](https://github.com/argoproj/argo-events/issues/26)
+ Inline, file and url sources for trigger workflows [#41](https://github.com/argoproj/argo-events/issues/41)

## v0.5-alpha1
+ Initial release
