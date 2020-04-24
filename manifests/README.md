# Argo Events Install Manifests

Several sets of manifests are provided:

| File | Description |
|------|-------------|
| [install.yaml](install.yaml) | Standard Argo Events cluster-wide installation. Gateway and sensor controllers operates on all namespaces |
| [namespace-install.yaml](namespace-install.yaml) | Installation of Argo Events which operates on a single namespace. Controller does not require to be run with clusterrole. Installs to `argo-events` namespace as an example. |

If installing with `kubectl install -f https://...`, remember to use the link to the file's raw version.
Otherwise you will get `mapping values are not allowed in this context`.

Manifests expect the namespace `argo-events` to exist. If you desire to deploy Argo events into a different namespace, change/overlay the namespace:

Cluster-wide install:

- In `Subject` section of [cluster-role-binding](cluster-install/rbac/argo-events-binding.yaml)

Namespace scope install:

- In ConfigMap overlays [gateway-controller-configmap](namespace-install/overlays/gateway-controller-configmap.yaml) and [sensor-controller-configmap](namespace-install/overlays/sensor-controller-configmap.yaml) to watch desired namespace

## Kustomize

You can use `./cluster-install` and `./namespace-install` as Kustomize remote bases.
