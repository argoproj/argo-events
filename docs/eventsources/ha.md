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
- Bitbucket
- Bitbucket Server
- GitHub
- GitLab
- NetApp Storage GRID
- Slack
- Stripe
- Webhook

When `spec.replicas` is set to N (N > 1), all the N Pods serve traffic.

## Active-Passive

If following EventSource types have `spec.replicas > 1`, `Active-Passive`
strategy is used, which means only one Pod serves traffic and the rest ones
stand by. One of standby Pods will be automatically elected to be active if the
old one is gone.

- AMQP
- Azure Events Hub
- Calendar
- Emitter
- GCP PubSub
- Generic
- File
- HDFS
- Kafka
- Minio
- MQTT
- NATS
- NSQ
- Pulsar
- Redis
- Resource

## Kubernetes Leader Election

By default, Argo Events will use NATS for the HA leader election except when
using a Kafka Eventbus, in which case a kubernetes leader election will be used.
If using a different EventBus you can opt-in to a Kubernetes native leader
election by specifying the following annotation.
```yaml
annotations:
  events.argoproj.io/leader-election: k8s
```

To use Kubernetes leader election the following RBAC rules need to be associated
with the EventSource ServiceAccount.
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: argo-events-leaderelection-role
rules:
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs:     ["get", "create", "update"]
```

## More

Click [here](../dr_ha_recommendations.md) to learn more information about Argo
Events DR/HA recommendations.
