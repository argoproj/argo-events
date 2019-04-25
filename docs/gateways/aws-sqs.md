# AWS SQS

AWS SNS gateway consumes messages from SQS queue.

### How to define an event source in confimap?
An entry in configmap corresponds to [this](https://github.com/argoproj/argo-events/blob/a913dafbf000eb05401ef2c847b29152af82977f/gateways/community/aws-sqs/config.go#L37-L51).

Because SQS works on polling, you need to provide a `waitTimeSeconds`.

## Setup

**1. Install [Gateway](../../examples/gateways/aws-sqs.yaml)**

Make sure gateway pod and service is running

**2. Install [Event Source](../../examples/event-sources/aws-sqs.yaml)**

**3. Install [Sensor](../../examples/sensors/aws-sqs.yaml)**

Make sure sensor pod is created.

**4. Trigger Workflow**

As soon as there a message is consumed from SQS queue, a workflow will be triggered.
