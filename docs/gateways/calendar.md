# Calendar

The gateway helps schedule K8s resources on an interval or on a cron schedule. It is solution for triggering any standard or custom K8s
resource instead of using CronJob.

## Setup
1. Create the [event Source](https://github.com/argoproj/argo-events/tree/master/examples/event-sources/calendar.yaml).

2. Deploy the [gateway](https://github.com/argoproj/argo-events/tree/master/examples/gateways/calendar.yaml).

3. Deploy the [sensor](https://github.com/argoproj/argo-events/tree/master/examples/sensors/calendar.yaml).
