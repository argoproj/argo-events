#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT=$(cd $(dirname "$0")/.. ; pwd)
CONTROLLER_TOOLS_PKG=${PROJECT_ROOT}/vendor/sigs.k8s.io/controller-tools
VERSION="v1alpha1"

# Sensor
go run ${CONTROLLER_TOOLS_PKG}/cmd/controller-gen/main.go crd:crdVersions=v1 \
    paths=github.com/argoproj/argo-events/pkg/apis/sensor/${VERSION} \
    output:stdout > ${PROJECT_ROOT}/manifests/base/crds/sensor-crd.yaml

# Gateway
go run ${CONTROLLER_TOOLS_PKG}/cmd/controller-gen/main.go crd:crdVersions=v1 \
    paths=github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION} \
    output:stdout > ${PROJECT_ROOT}/manifests/base/crds/gateway-crd.yaml

# EventSource
go run ${CONTROLLER_TOOLS_PKG}/cmd/controller-gen/main.go crd:crdVersions=v1 \
    paths=github.com/argoproj/argo-events/pkg/apis/eventsources/${VERSION} \
    output:stdout > ${PROJECT_ROOT}/manifests/base/crds/eventsources-crd.yaml
