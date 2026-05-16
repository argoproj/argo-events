# EventBus

![GA](../assets/ga.svg)

> v0.17.0 and after

EventBus is a Kubernetes
[Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
which is used for event transmission from EventSources to Sensors. Currently,
EventBus is backed by [NATS](https://docs.nats.io/), including both their NATS
Streaming service, their newer Jetstream service, and Kafka. In the future,
this can be expanded to support other technologies as well.

EventBus is namespaced; an EventBus object is required in a namespace to make
EventSource and Sensor work.

The common practice is to create an EventBus named `default` in the namespace. If
you want to use a different name, or you want to have multiple EventBus in one
namespace, you need to specify `eventBusName` in the spec of EventSource and
Sensor correspondingly, so that they can find the right one. See EventSource
[spec](../APIs.md#argoproj.io/v1alpha1.EventSourceSpec)
and Sensor
[spec](../APIs.md#argoproj.io/v1alpha1.SensorSpec).

## Cross-namespace EventBus reference (Sensor)

By default a Sensor looks for its EventBus in its own namespace. Platform teams
that keep a centralised EventBus (plus its Secrets) in a shared namespace can
instruct a Sensor to subscribe to it by setting `eventBusNamespace`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: my-sensor
  namespace: app-ns          # Sensor lives here
spec:
  eventBusName: default
  eventBusNamespace: bus-ns  # EventBus lives here
  # … dependencies, triggers …
```

### Requirements and limitations

**Secrets must be pre-replicated.** Any Secret referenced by the EventBus
(access token, TLS certificates) must also exist in the Sensor's namespace.
The controller does not copy Secrets across namespaces. A tool such as
[Kyverno](https://kyverno.io/) can automate the replication.

**Cluster-scoped install only.** The cross-namespace lookup requires the
controller to read EventBus objects outside its own namespace. This is only
possible with the cluster-scoped installation profile. If you use the
namespace-scoped install (`--namespaced`), ensure the EventBus namespace is
included in `--managed-namespace`. See [Managed Namespace](../managed-namespace.md)
for details.

**EventSource cross-namespace reference is not yet supported.** Only the Sensor
side is implemented. A follow-up contribution can mirror the same change for
EventSource.
