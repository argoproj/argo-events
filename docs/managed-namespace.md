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
