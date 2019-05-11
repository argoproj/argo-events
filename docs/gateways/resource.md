# Resource

Resource gateway listens to updates on **any** Kubernetes resource.

## Setup
1. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/resource.yaml).

2. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/resource.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/resource.yaml).

