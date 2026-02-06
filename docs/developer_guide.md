# Developer Guide

## Setup your DEV environment

Argo Events is native to Kubernetes so you'll need a running Kubernetes cluster.
This guide includes steps for `Minikube` for local development, but if you have
another cluster you can ignore the Minikube specific step 3.

### Requirements

- Golang 1.25.3+
- Docker

### Installation & Setup

#### 1. Get the project

```
git clone git@github.com:argoproj/argo-events
cd argo-events
```

#### 2. Start Minikube and point Docker Client to Minikube's Docker Daemon

```
minikube start
eval $(minikube docker-env)
```

#### 3. Build the project

```
make build
```

### 4. Changing Types

If you're making a change to the `pkg/apis` package, please ensure you re-run the
following command for code regeneration.

```
make codegen
```

### Deploying and testing your changes on a local dev cluster

Make sure you have a local dev cluster running and have the current kubeconfig context set to the local cluster as the following steps will use that to load the images and deploy the controller. The local dev cluster can be Minikube, Kind, or K3D.

#### 1. Building images

If you want a fresh build before building the container image, run the following commands to delete any existing binaries.

```
rm -rf dist
```

The following command will build the images and load/import it into the local cluster.

```
make image
```

#### 2. Make the manifests

To generate the manifests, run the following command:

```
make manifests
```

#### 3. Deploy the custom resource definitions (CRDs) and custom build of the controller

Run the following command to deploy the CRDs and controller:

```
make start
```

> ![NOTE]
> This command will build the images so you don't need to run `make image` before running `make start`.

### Deploying and testing your changes on managed clusters in the cloud

You might want to test certain changes on a managed cluster in the cloud. For example, testing Microsoft Entra ID Workload Identity on AKS. The following steps will guide you through that process.

#### 1. Building images

We can use ephemeral containers for quick testing with images hosted by our friends at [ttl.sh](https://ttl.sh/). 

If you want a fresh build before building the container image, run the following commands to delete any existing binaries.

```
rm -rf dist
```

The ephemeral container image repository must be unique and the tag is used to determine how long the image is kept.

Run the following commands to generate a unique repository name and tag it so the image will be kept for 1 hour.

```
IMG_REPO=$(uuidgen | tr '[:upper:]' '[:lower:]')
IMG_VERSION=1h
```

Using the ephemeral image repository and version, run the following command to build and push a multi-arch image to ttl.sh.

```
DOCKER_PUSH=true IMAGE_NAMESPACE=ttl.sh/$IMG_REPO VERSION=$IMG_VERSION make image-multi
```

#### 2. Make the manifests

To generate the manifests, run the following command:

```
make manifests
```

#### 3. Deploy the custom resource definitions (CRDs) and custom build of the controller

To deploy the CRDs and controller, we can create a kustomization.yaml file for the cluster based installation and customize it to use the custom image.

Run the following command to create a kustomization.yaml file for the cluster based installation.

```
kustomize create --resources manifests/install.yaml
```

Add the image to the kustomization.yaml file by running the following command:

```
cat <<EOF >> kustomization.yaml
images:
  - name: quay.io/argoproj/argo-events
    newName: ttl.sh/${IMG_REPO}/argo-events
    newTag: ${IMG_VERSION}
patches:
  - target:
      kind: Deployment
      name: controller-manager
      namespace: argo-events
    patch: |-
      - op: remove
        path: /spec/template/spec/containers/0/env/0
      - op: add
        path: /spec/template/spec/containers/0/env/0
        value:
          name: ARGO_EVENTS_IMAGE
          value: ttl.sh/${IMG_REPO}/argo-events:${IMG_VERSION}
EOF
```

Make sure you are connected to the cluster you want to deploy to then run the following command to create a namespace for Argo Events.

```
kubectl create namespace argo-events
```

Deploy Argo Events with the custom container image to the cluster.

```
kustomize build . | kubectl apply -f -
```

Now you can test your changes.

#### 4. Uninstalling custom deployment of Argo Events

To uninstall Argo Events, run the following commands:

```
kustomize build . | kubectl delete -f -
kubectl delete namespace argo-events --wait=false
rm kustomization.yaml
```