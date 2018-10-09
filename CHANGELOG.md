# Changelog

## v0.6-alpha1
+ Custom gateways for Slack, Twitter, AWS S3, Google Cloud Pub-Sub, Reddit and Docker
+ FileWatcher as core gateway
+ Kubernetes configmaps as artifact locations for triggers
+ Tutorials on setting up pipelines

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
