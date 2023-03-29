# Sensor High Availability

Sensor controller creates a k8s deployment (replica number defaults to 1) for
each Sensor object. HA with `Active-Passive` strategy can be achieved by setting
`spec.replicas` to a number greater than 1, which means only one Pod serves
traffic and the rest ones stand by. One of standby Pods will be automatically
elected to be active if the old one is gone.

**Please DO NOT manually scale up the replicas, that might cause unexpected
behaviors!**

## Kubernetes Leader Election

By default, Argo Events will use NATS for the HA leader election except when
using a Kafka Eventbus, in which case a leader election is not required as a
Sensor that uses a Kafka EventBus is capable of horizontally scaling. If using
a different EventBus you can opt-in to a Kubernetes native leader election by
specifying the following annotation.
```yaml
annotations:
  events.argoproj.io/leader-election: k8s
```

To use Kubernetes leader election the following RBAC rules need to be associated
with the Sensor ServiceAccount.
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
