# Trigger Guide
Triggers are the sensor's actions. Triggers are only executed after all of the sensor's signals have been resolved.

The `resource` field in the trigger object has details of what to execute when the signals have been resolved. The `source` field in the `resource` object can have 3 types of values:

- inline:
In this case, the workflow to execute as part of the trigger is inlined in the sensor yaml itself. E.g. [inline-sensor](https://github.com/argoproj/argo-events/blob/master/examples/inline-sensor.yaml)

- file:
In this case, the workflow to execute is specified as a file-system path. This file-system path should exist in the sensor-controller deployment. The default sensor-controller does not have any volume mounts and therefore does not have any workflow yamls. If users are going to use this, they should explicitly mount appropriate volumes in the sensor-controller deployment. E.g. [file-sensor](https://github.com/argoproj/argo-events/blob/master/examples/file-sensor.yaml)

- url:
In this case, the workflow to execute is specified as a url path. E.g. [url-sensor](https://github.com/argoproj/argo-events/blob/master/examples/url-sensor.yaml)


### Resource Object
Resources define a YAML or JSON K8 resource. The set of currently resources supported are implemented in the `store` package. Adding support for new resources is as simple as including the type you want to create in the store's `decodeAndUnstructure()` method. We hope to change this functionality so that permissions for CRUD operations against certain resources can be controlled through RBAC roles instead.

List of currently supported K8 Resources:
- Sensor
- [Workflow](https://github.com/argoproj/argo)

### Messages
Messages define content and a stream queue resource on which to send the content. 
