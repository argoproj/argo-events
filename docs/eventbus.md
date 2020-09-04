# EventBus

![alpha](assets/alpha.svg)

> v0.17.0 and after

EventBus is a kubernetes
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
[spec](https://github.com/argoproj/argo-events/tree/stable/api/event-source.md#eventsourcespec) and Sensor
[spec](https://github.com/argoproj/argo-events/tree/stable/api/sensor.md#sensorspec).

## NATS Streaming

You can create a `native` NATS EventBus, or connect to an existing NATS
Streaming service with `exotic` NATS EventBus.

### Native

A simplest `native` NATS EeventBus example:

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
persistent volumes, the StatefulSet PODs will be created with anti-affinity
rule.

```yaml
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      replicas: 3 # optional, defaults to 3, and requires minimal 3
      auth: token # optional, default to none
      antiAffinity: true # optional, default to false
      persistence: # optional
        storageClassName: standard
        accessMode: ReadWriteOnce
        volumeSize: 10Gi
```

- `replicas` - StatefulSet replicas, defaults to 3, and requires minimal 3.
  According to
  [NATS Streaming doc](https://docs.nats.io/nats-streaming-concepts/clustering),
  the size should probably be limited to 3 to 5, and odd number is recommended.

- `auth` - The strategy that clients connect to NATS Streaming service, `none`
  or `token` is currently supported, defaults to `none`.

  If `token` strategy is used, the system will generate a token and store it in
  K8s secrets (one for client, one for server), EventSource and Sensor PODs will
  automatically load the client secret and use it to connect to the EventBus.

- `antiAffinity` - Whether to create the PODs with anti-affinity rule.

- `persistence` - Whether to use a
  [persistence volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
  for the data.

- Check [here](https://github.com/argoproj/argo-events/tree/stable/api/event-bus.md#argoproj.io/v1alpha1.NativeStrategy) for the
  full spec.

#### More About Native NATS EventBus

- Messages limit is 1,000,000.

- Max age of messages is 72 hours, which means messages over 72 hours will be
  deleted automatically.

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
