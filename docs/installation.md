# Installation

### Requirements

* Kubernetes cluster >=v1.11
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.11.0

### Using kubectl

#### Cluster-wide Installation

1. Create the namespace

        kubectl create namespace argo-events

2. Deploy Argo Events, SA, ClusterRoles, Sensor Controller, EventBus Controller and EventSource Controller

        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/install.yaml

       NOTE: 
       
         * On GKE, you may need to grant your account the ability to create new custom resource definitions and clusterroles
    
                kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
           
         * On Openshift, make sure to grant `anyuid` scc to the service account.

                oc adm policy add-scc-to-user anyuid system:serviceaccount:argo-events:default

3. Deploy the eventbus,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml

#### Namespace Installation

1. Create the namespace

        kubectl create namespace argo-events

2. Deploy Argo Events, SA, Roles, Sensor Controller, EventBus Controller and EventSource Controller

        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/manifests/namespace-install.yaml

       NOTE: 
       
         * On GKE, you may need to grant your account the ability to create new custom resource definitions
    
                kubectl create clusterrolebinding YOURNAME-cluster-admin-binding --clusterrole=cluster-admin --user=YOUREMAIL@gmail.com
         
         * On Openshift, make sure to grant `anyuid` scc to the service account.

                oc adm policy add-scc-to-user anyuid system:serviceaccount:argo-events:default

3. Deploy the eventbus,

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/eventbus/native.yaml        


### Using Kustomize

Use either [`cluster-install`](https://github.com/argoproj/argo-events/tree/stable/manifests/cluster-install) or [`namespace-install`](https://github.com/argoproj/argo-events/tree/stable/manifests/namespace-install) folder as your base for Kustomize.

`kustomization.yaml`:

    bases:
      - github.com/argoproj/argo-events/manifests/cluster-install
      # OR
      - github.com/argoproj/argo-events/manifests/namespace-install

### Using Helm Chart

Note: This method does not work with Helm 3, only Helm 2.

Make sure you have helm client installed and Tiller server is running. To install helm, follow <a href="https://docs.helm.sh/using_helm/">the link.</a>

1. Create namespace called argo-events.

1. Add `argoproj` repository

        helm repo add argo https://argoproj.github.io/argo-helm

1. The helm chart for argo-events is maintained solely by the community and hence the image version for controllers can go out of sync.
   Update the image version in values.yaml to v0.17.0.

1. Install `argo-events` chart

        helm install argo-events argo/argo-events

### Migrate to v0.17.0

If you are looking to migrate Argo Events <0.16.0 to v0.17.0, please read the [migration docs](https://github.com/argoproj/argo-events/wiki/Migration-path-for-v0.17.0).
