# Slack

The gateway listens to events from Slack.
The gateway will not register the webhook endpoint on Slack. You need to manually do it.

## Setup

1. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/slack.yaml).

2. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/slack.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/slack.yaml).

## Trigger Workflow
A workflow will be triggered when slack sends an event.
