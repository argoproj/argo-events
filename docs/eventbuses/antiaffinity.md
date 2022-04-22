## Anti-affinity

Kubernetes offers a concept of [anti-affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/), meaning that pods are scheduled on separate nodes. The anti-affinity can either be "best effort" or a hard requirement.

A best effort and a hard requirement node anti-affinity config look like
  below, if you want to do AZ (Availability Zone) anti-affinity, change the value
  of `topologyKey` from `kubernetes.io/hostname` to
  `topology.kubernetes.io/zone`.

```yaml
# Best effort
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

```yaml
# Hard requirement
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            controller: eventbus-controller
            eventbus-name: default
        topologyKey: kubernetes.io/hostname
```
