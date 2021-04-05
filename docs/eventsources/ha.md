# EventSource High Availability

EventSource controller creates a k8s deployment (replica number defaults to 1)
for each EventSource object to watch the events. HA can be achieved by setting
`spec.replicas` to a number greater than 1.

Some types of the event sources do not allow multiple live clients with same
attributes (i.e. multiple clients with same `clientID` connecting to a NATS
server), or multiple event source PODs will generate duplicated events to
downstream, so the HA strategies are different for different event sources.

**Please DO NOT manually scale up the replicas, that might cause unexpected
behaviors!**

## Active-Active

`Active-Active` strategy is applied to the following EventSource types.

- AWS SNS
- AWS SQS
- Github
- Gitlab
- NetApp Storage GRID
- Resource
- Slack
- Stripe
- Webhook

When `spec.replicas` is set to N (N > 1), all the N Pods serve trafic.

## Active-Passive

If following EventSource types have `spec.replicas > 1`, `Active-Passive`
strategy is used, which means only one Pod serves traffic and the rest ones
stand by. One of standby Pods will be automatically elected to be active if the
old one is gone.

- AMQP
- Azure Events Hub
- Kafka
- GCP PubSub
- File
- HDFS
- NATS
- Minio
- MQTT
- Emitter
- NSQ
- Pulsar
- Redis
- Calendar
- Generic

### RBAC

To achieve `Active-Passive` strategy for these EventSources, a Service Account
with extra RBAC settings is needed. The Service Account needs to be bound to a
Role like following, and specified in the spec through
`spec.template.serviceAccountName`.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: lease-role
rules:
  - apiGroups:
      - coordination.k8s.io
    resources:
      - leases
    resourceNames:
      - eventsource-{event-source-name}
    verbs:
      - "*"
```

**NOTE: This is not requried if `spec.replicas = 1`.**

## More

Check [this](../dr_ha_recommendations.md) out to learn more information about
DR/HA.
