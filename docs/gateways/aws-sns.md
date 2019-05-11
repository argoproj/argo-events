# AWS SNS

The gateway listens to notifications from AWS SNS.

## Why is there webhook in the gateway?
Because one of the ways you can receive notifications from SNS is over http. So, the gateway runs a http server internally.
Once you create an entry in the event source configmap, the gateway will register the url of the server on AWS.
All notifications for that topic will then be dispatched by SNS over to the endpoint specified in event source.

The gateway spec defined in `examples` has a `serviceSpec`. This service is used to expose the gateway server to the outside world.

## How to get the URL for the service?
Depending upon the Kubernetes provider, you can create the Ingress or Route. 

## Setup

1. Deploy [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/aws-sns.yaml) before creating event sources because you need to have the gateway pod running and a service backed by the pod, so that you can get the URL for the service. 

2. Create the [event source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/aws-sns.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/aws-sns.yaml).

## Trigger Workflow

As soon as a message is published on your SNS topic, a workflow will be triggered.
 