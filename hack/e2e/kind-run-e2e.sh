#!/bin/bash

set -e

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/../..; pwd)
KUBERNETES_VERSION=${KUBERNETES_VERSION:-kindest/node:v1.13.4}
CLUSTER_NAME=${CLUSTER_NAME:-kind-argo-events}
IMAGE_PREFIX=${IMAGE_PREFIX:-argoproj/}
IMAGE_TAG=${IMAGE_TAG:-v0.11}

kind create cluster --name $CLUSTER_NAME --image $KUBERNETES_VERSION
export KUBECONFIG="$(kind get kubeconfig-path --name=$CLUSTER_NAME)"
kubectl cluster-info
kind load docker-image --name $CLUSTER_NAME ${IMAGE_PREFIX}sensor-controller:${IMAGE_TAG} ${IMAGE_PREFIX}gateway-controller:${IMAGE_TAG} ${IMAGE_PREFIX}webhook-gateway:${IMAGE_TAG} ${IMAGE_PREFIX}gateway-client:${IMAGE_TAG}

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/../..; pwd)

echo "* Set up e2e test"

kubectl create namespace argo-events
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/k8s/manifests/argo-events-sa.yaml
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/k8s/manifests/argo-events-cluster-roles.yaml
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/k8s/manifests/sensor-crd.yaml
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/k8s/manifests/gateway-crd.yaml
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/k8s/manifests/sensor-controller-configmap.yaml
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/k8s/manifests/gateway-controller-configmap.yaml

# changes are only made for controller images
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/e2e/manifests/sensor-controller-deployment.yaml
kubectl apply -n argo-events -f $PROJECT_ROOT/hack/e2e/manifests/gateway-controller-deployment.yaml


# wait for controllers to get up and running
sleep 10

echo "* Run e2e tests."
go test -v ./test/e2e/...

# delete the cluster
kind delete cluster --name $CLUSTER_NAME
