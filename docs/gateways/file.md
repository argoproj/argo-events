# File

File gateway watches changes to the file within specified directory.

## Where the directory should be?
The directory can be in the pod's own filesystem or you can mount a persistent volume and refer to a directory.
Make sure that the directory exists before you create the gateway configmap.

## How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/core/file/config.go#L34-L38).

### Event Payload Structure

[Structure](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/common/fsevent/fileevent.go#L11-L14)  


## Setup
**1. Install [Gateway Configmap](../../../examples/gateways/file-gateway-configmap.yaml)**

**2. Install [Gateway](../../../examples/gateways/file.yaml)**

Make sure the gateway pod is created.

**3. Install [Sensor](../../../examples/sensors/file.yaml)**

Make sure the sensor pod is created.

**4. Trigger Workflow**

Exec into the gateway pod and go to the directory specified in event source and create a file. That should generate an event causing sensor to trigger a workflow.


## How to listen to notifications from different directories
Simply edit the gateway configmap and add new entry that contains the configuration required to listen to file within different directory and save
the configmap. The gateway will start listening to file notifications from new directory as well.
