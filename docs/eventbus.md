# EventBus

![GA](assets/ga.svg)

> v0.17.0 and after

EventBus is a Kubernetes
[Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
which is used for events transmission from EventSources to Sensors. Currently
EventBus is backed by
[NATS Streaming](https://github.com/nats-io/nats-streaming-server), and it is
open to support other technologies.

EventBus is namespaced, an EventBus object is required in a namespace to make
EventSource and Sensor work.

The common pratice is to create an EventBus named `default` in the namespace. If
you want to use a different name, or you want to have multiple EventBus in one
namespace, you need to specifiy `eventBusName` in the spec of EventSource and
Sensor correspondingly, so that they can find the right one. See EventSource
[spec](https://github.com/argoproj/argo-events/tree/stable/api/event-source.md#eventsourcespec)
and Sensor
[spec](https://github.com/argoproj/argo-events/tree/stable/api/sensor.md#sensorspec).

## NATS Streaming

You can create a `native` NATS EventBus, or connect to an existing NATS
Streaming service with `exotic` NATS EventBus.

### Native

A simplest `native` NATS EventBus example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native: {}
```

The example above brings up a NATS Streaming
[StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
with 3 replicas in the namespace.

The following example shows an EventBus with `token` auth strategy and
persistent volumes.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      replicas: 3 # optional, defaults to 3, and requires minimal 3
      auth: token # optional, default to none
      persistence: # optional
        storageClassName: standard
        accessMode: ReadWriteOnce
        volumeSize: 10Gi
```

#### Properties

Check
[here](https://github.com/argoproj/argo-events/tree/stable/api/event-bus.md#argoproj.io/v1alpha1.NativeStrategy)
for the full spec of `native`.

- `replicas` - StatefulSet replicas, defaults to 3, and requires minimal 3.
  According to
  [NATS Streaming doc](https://docs.nats.io/nats-streaming-concepts/clustering),
  the size should probably be limited to 3 to 5, and odd number is recommended.

- `auth` - The strategy that clients connect to NATS Streaming service, `none`
  or `token` is currently supported, defaults to `none`.

  If `token` strategy is used, the system will generate a token and store it in
  K8s secrets (one for client, one for server), EventSource and Sensor PODs will
  automatically load the client secret and use it to connect to the EventBus.

- `antiAffinity` - Whether to create the StatefulSet PODs with anti-affinity
  rule. **Deprecated** in `v1.3`, will be removed in `v1.5`, use `affinity`
  instead.

- `nodeSelector` -
  [Node selector](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
  for StatefulSet PODs.

- `tolerations` -
  [Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
  for the PODs.

- `persistence` - Whether to use a
  [persistence volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
  for the data.

- `securityContext` - POD level
  [security attributes](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)
  and common container settings.

- `maxAge` - Max Age of existing messages, i.e. `72h`, `4h35m`, defaults to
  `72h`.

- `imagePullSecrets` - Secrets used to pull images.

- `serviceAccountName` - In case your firm requires to use a service account
  other than `default`.

- `priority` -
  [Priority](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/)
  of the StatefulSet PODs.

- `priorityClassName` -
  [PriorityClassName](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/)
  of the StatefulSet PODs.

- `affinity` -
  [Affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/)
  settings for the StatefulSet PODs.

  A best effort and a hard requirement node anti-affinity config look like
  below, if you want to do AZ (Availablity Zone) anti-affinity, change the value
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

#### More About Native NATS EventBus

- Messages limit per channel defaults to 1,000,000. It could be customized by
  setting `spec.nats.native.maxMsgs`, `0` means unlimited.

- Message bytes per channel defaults to `1GB`, setting
  `spec.nats.native.maxBytes` to customize it, `"0"` means unlimited.

- Max age of messages is 72 hours, which means messages over 72 hours will be
  deleted automatically. It can be cutomized by setting
  `spec.nats.native.maxAge`, i.e. `240h`.

- Max subscription number is 1000.

### Exotic

To use an existing NATS Streaming service, follow the example below.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    exotic:
      url: nats://xxxxx:xxx
      clusterID: cluster-id
      auth: token
      accessSecret:
        name: my-secret-name
        key: secret-key
```

## More Information

- To view a finalized EventBus config:

```sh
kubectl get eventbus default  -o json | jq '.status.config'
```

A sample result:

```json
{
  "nats": {
    "accessSecret": {
      "key": "client-auth",
      "name": "eventbus-default-client"
    },
    "auth": "token",
    "clusterID": "eventbus-default",
    "url": "nats://eventbus-default-stan-svc:4222"
  }
}
```

- All the events in a namespace are published to same channel/subject/topic
  named `eventbus-{namespace}` in the EventBus.
