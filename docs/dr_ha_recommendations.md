# HA/DR Recommendations

## EventBus

A simple EventBus used for non-prod deployment or testing purpose could be:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      auth: token
```

However this is not good enough to run your production deployment, following
settings are recommended to make it more reliable, and achieve high
availability.

### Persistent Volumes

Even though the EventBus PODs already have data sync mechanism between them,
persistent volumes are still recommended to be used to avoid any events data
lost when the PODs crash.

An EventBus with persistent volumes looks like below:

```yaml
spec:
  nats:
    native:
      auth: token
      persistence:
        storageClassName: standard
        accessMode: ReadWriteOnce
        volumeSize: 20Gi
```

### Anti-Affinity

You can run the EventBus PODs with anti-affinity, to avoid the situation that
all PODs are gone when a disaster happens.

An EventBus with best effort node anti-affinity:

```yaml
spec:
  nats:
    native:
      auth: token
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - podAffinityTerm:
                labelSelector:
                  matchLabels:
                    controller: eventbus-controller
                    eventbus-name: default
                topologyKey: kubernetes.io/hostname
              weight: 100
```

An EventBus with hard requirement node anti-affinity:

```yaml
spec:
  nats:
    native:
      auth: token
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  controller: eventbus-controller
                  eventbus-name: default
              topologyKey: kubernetes.io/hostname
```

To do AZ (Availablity Zone) anti-affinity, change the value of `topologyKey`
from `kubernetes.io/hostname` to `topology.kubernetes.io/zone`.

Besides `affinity`,
[nodeSelector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector)
and
[tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
also could be set through `spec.nats.native.nodeSelector` and
`spec.nats.native.tolerations`.

### POD Priority

Setting
[POD Prioryty](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/)
could reduce the chance of PODs being evicted.

Priority could be set through `spec.nats.native.priorityClassName` or
`spec.nats.native.priority`.

## EventSources

### Replicas

For below types of EventSources, `spec.replica` could be set to a number `>1` to
make them HA, see more detail [here](eventsources/deployment-strategies.md).

- AWS SNS
- AWS SQS
- Github
- Gitlab
- NetApp Storage GRID
- Slack
- Stripe
- Webhook

### EventSource POD Node Selection

EventSource POD `affinity`, `nodeSelector` and `tolerations` could be set
through `spec.template.affinity`, `spec.template.nodeSelector` and
`spec.template.tolerations`.

### EventSource POD Priority

Priority could be set through `spec.template.priorityClassName` or
`spec.template.priority`.

## Sensors

### Sensor POD Node Selection

Sensor POD `affinity`, `nodeSelector` and `tolerations` could also be set
through `spec.template.affinity`, `spec.template.nodeSelector` and
`spec.template.tolerations`.

### Sensor POD Priority

Priority could be set through `spec.template.priorityClassName` or
`spec.template.priority`.
