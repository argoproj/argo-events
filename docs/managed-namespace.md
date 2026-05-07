# Managed Namespace

You can install `argo-events` in either cluster scoped or namespace scoped configuration, accordingly you need to set up ClusterRole or normal Role for service account `argo-events-sa`.

## v1.7+

In namespace scope installation, you must run `controller-manager` deployment with `--namespaced`. If you would like to have the controller watching a separate namespace, add `--managed-namespace` as well.

For example:

```
      - args:
        - --namespaced
        - --managed-namespace
        - default
```

## Prior to v1.7

There were 3 controller deployments (`eventbus-controller`, `eventsource-controller` and `sensor-controller`) in the versions prior to v1.7, to run namespaced installation, add `--namespaced` argument to each of them. Argument `--managed-namespace` is also supported to watch a different namespace.

## Cross-namespace EventBus references

When a Sensor specifies `eventBusNamespace` to subscribe to an EventBus in a
different namespace, the controller must be able to read that EventBus object.

- **Cluster-scoped install** — works out of the box; the ClusterRole already
  grants `get`/`list`/`watch` on `eventbusses` cluster-wide.
- **Namespace-scoped install** — the controller can only read EventBuses inside
  its managed namespace. To use cross-namespace references, include the EventBus
  namespace in `--managed-namespace`, or switch to the cluster-scoped install.

See [Cross-namespace EventBus reference](eventbus/eventbus.md#cross-namespace-eventbus-reference-sensor)
for usage details.
