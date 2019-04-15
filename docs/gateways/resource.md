# Resource

Resource gateway listens to updates on **any** Kubernetes resource.

You consider using resource gateway as the watchdog in your cluster, basically monitoring various updated to 
installed kubernetes resources. 

## How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/ebdbdd4a2a8ce47a0fc6e9a6a63531be2c26148a/gateways/core/resource/config.go#L36-L46).

### Example
The [gateway configmap](../../../examples/gateways/resource-gateway-configmap.yaml) contains a couple of event sources. 

The `workflow-success` event source defines configuration to listen to notifications for newly created `Argo Workflows` and filter these
workflows on labels. 

The `namespace` event source defines configuration to listen to any updates to namespace.

### Event Payload Structure
Kubernetes Object the gateway is watching.

## Setup
**1. Install [Gateway Configmap](../../examples/gateways/resource-gateway-configmap.yaml)**

**2. Install [Gateway](../../examples/gateways/resource.yaml)**

Make sure the gateway pod is created.

**3. Install [Sensor](../../examples/sensors/resource.yaml)**

Make sure the sensor pod is created.

**4. Trigger workflow**

Create an argo workflow with name `my-workflow`. As soon as `my-workflow` succeeds, a new workflow will be triggered.

