# AWS SNS

AWS SNS gateway listens to notifications from SNS.

### How to define an event source in confimap?
An entry in the gateway configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/community/aws-sns/config.go#L70-L75).

### Why is there webhook in the gateway?
Because one of the ways you can receive notifications from SNS is over http. So the gateway runs a http server internally.
Once you create a new gateway configmap or add a new event source entry in the configmap, the gateway will register the url of the server on AWS.
All notifications for that topic will then be dispatched by SNS over to the endpoint specified in event source.

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server to the outside world.

**How to get the URL for the service?**
Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

**1. Install [Gateway](../../../examples/gateways/aws-sns.yaml)**

We are installing gateway before creating configmap. Because we need to have the gateway pod running and a service backed by the pod, so 
that we can get the URL for the service. 

Make sure gateway pod and service is running

**2. Install [Gateway Configmap](../../../examples/gateways/aws-sns-gateway-configmap.yaml)**

**3. Install [Sensor](../../../examples/sensors/aws-sns.yaml)**

Make sure sensor pod is created.

**4. Trigger Workflow**

As soon as there is message on your SNS topic, a workflow will be triggered.
 