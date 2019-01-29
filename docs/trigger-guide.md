# Triggers
Triggers are the sensor's actions. Triggers are only executed after all of the sensor's signals have been resolved.

The `resource` field in the trigger object has details of what to execute when the signals have been resolved. Refer to https://github.com/argoproj/argo-events/blob/master/docs/artifact-guide.md
### Resource Object
Resources define a YAML or JSON K8 resource. The set of currently resources supported are implemented in the `store` package. Adding support for new resources is as simple as including the type you want to create in the store's `decodeAndUnstructure()` method. We hope to change this functionality so that permissions for CRUD operations against certain resources can be controlled through RBAC roles instead.

List of currently supported custom K8 Resources:
- Gateway
- Sensor
- [Workflow](https://github.com/argoproj/argo)

