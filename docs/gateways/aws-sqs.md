# AWS SQS

The gateway consumes messages from AWS SQS queue.

## Setup

1. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/aws-sqs.yaml)

2. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/aws-sqs.yaml).  Because SQS works on polling, you need to provide a `waitTimeSeconds`.


3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/aws-sqs.yaml).

## Trigger Workflow
As soon as there a message is consumed from SQS queue, a workflow will be triggered.
