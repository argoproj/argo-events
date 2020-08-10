# Managed Namespace

You can install `argo-events` in either cluster scoped or namespace scoped
configuration, accordingly you need to set up ClusterRole or normal Role for
service account `argo-events-sa`.

In namespace scope installation, you must run `eventbus-controller`,
`eventsource-controller` and `sensor-controller` with `--namespaced`. If you
would like to have the controllers watching a separated namespace, add
`--managed-namespace` as well.

For example:

```
      - args:
        - --namespaced
        - --managed-namespace
        - default
```
