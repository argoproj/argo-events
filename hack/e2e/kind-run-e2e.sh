#!/bin/bash

set -e

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/../..; pwd)
KUBERNETES_VERSION=${KUBERNETES_VERSION:-kindest/node:v1.13.4}
CLUSTER_NAME=${CLUSTER_NAME:-kind-argo-events}
IMAGE_PREFIX=${IMAGE_PREFIX:-argoproj/}
IMAGE_TAG=${IMAGE_TAG:-latest}

kind create cluster --name $CLUSTER_NAME --image $KUBERNETES_VERSION
export KUBECONFIG="$(kind get kubeconfig-path)"

function cleanup {
  if [[ -z "$KEEP_CLUSTER" ]]; then
    echo "* Cleaning up the e2e cluter..."
    kind delete cluster --name $CLUSTER_NAME
  else
    echo "* Skip e2e cluster cleanup for $CLUSTER_NAME."
  fi
}
trap cleanup EXIT

kind load docker-image --name $CLUSTER_NAME ${IMAGE_PREFIX}sensor-controller:${IMAGE_TAG} ${IMAGE_PREFIX}gateway-controller:${IMAGE_TAG} ${IMAGE_PREFIX}webhook-gateway:${IMAGE_TAG} ${IMAGE_PREFIX}gateway-client:${IMAGE_TAG}

# Avoid too early access
sleep 10

$PROJECT_ROOT/hack/e2e/run-e2e.sh
