# Installation

### Requirements

* Kubernetes cluster >=v1.11
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.11.0

### Using kubectl

#### Cluster-wide Installation

1. Create the namespace.

        kubectl create namespace argo-events

2. Deploy Argo Events SA, ClusterRoles, and Controller for Sensor, EventBus, and EventSource.

        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml
        # Install with a validating admission controller
        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install-validating-webhook.yaml


       NOTE:

         * On GKE, you may need to grant your account the ability to create new custom resource definitions and clusterroles

                kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com

         * On OpenShift:
             - Make sure to grant `anyuid` scc to the service accounts.

                oc adm policy add-scc-to-user anyuid system:serviceaccount:argo-events:argo-events-sa system:serviceaccount:argo-events:argo-events-webhook-sa

             - Add update permissions for the `deployments/finalizers` and `clusterroles/finalizers` of the argo-events-webhook ClusterRole (this is necessary for the validating admission controller)

                - apiGroups:
                  - rbac.authorization.k8s.io
                  resources:
                  - clusterroles/finalizers
                  verbs:
                  - update
                - apiGroups:
                  - apps
                  resources:
                  - deployments/finalizers
                  verbs:
                  - update

3. Deploy the eventbus.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

#### Namespace Installation

1. Create the namespace.

        kubectl create namespace argo-events

2. Deploy Argo Events SA, ClusterRoles, and Controller for Sensor, EventBus, and EventSource.

        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/namespace-install.yaml

       NOTE:

         * On GKE, you may need to grant your account the ability to create new custom resource definitions

                kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com

         * On OpenShift:
             - Make sure to grant `anyuid` scc to the service account.

                oc adm policy add-scc-to-user anyuid system:serviceaccount:argo-events:default

             - Add update permissions for the `deployments/finalizers` and `clusterroles/finalizers` of the argo-events-webhook ClusterRole (this is necessary for the validating admission controller)

                - apiGroups:
                  - rbac.authorization.k8s.io
                  resources:
                  - clusterroles/finalizers
                  verbs:
                  - update
                - apiGroups:
                  - apps
                  resources:
                  - deployments/finalizers
                  verbs:
                  - update

3. Deploy the eventbus.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

### Using Kustomize

Use either [`cluster-install`](https://github.com/argoproj/argo-events/tree/stable/manifests/cluster-install), or [`cluster-install-with-extension`](https://github.com/argoproj/argo-events/tree/stable/manifests/cluster-install-with-extension), or [`namespace-install`](https://github.com/argoproj/argo-events/tree/stable/manifests/namespace-install) folder as your base for Kustomize.

`kustomization.yaml`:

    bases:
      - github.com/argoproj/argo-events/manifests/cluster-install
      # OR
      - github.com/argoproj/argo-events/manifests/namespace-install

### Using Helm Chart

Make sure you have the helm client installed. To install helm, follow <a href="https://docs.helm.sh/using_helm/">the link.</a>

1. Add `argoproj` repository.

        helm repo add argo https://argoproj.github.io/argo-helm

1. The helm chart for argo-events is maintained solely by the community and hence the image version for controllers can go out of sync.
   Update the image version in values.yaml to v1.0.0.

1. Install `argo-events` chart.

        helm install argo-events argo/argo-events -n argo-events --create-namespace

1. Deploy the eventbus.

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

### Migrate to v1.0.0

If you are looking to migrate Argo Events <0.16.0 to v1.0.0, please read the [migration docs](https://github.com/argoproj/argo-events/wiki/Migration-path-for-v0.17.0).
