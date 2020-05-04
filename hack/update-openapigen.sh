#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
CODEGEN_PKG=${PROJECT_ROOT}/vendor/k8s.io/kube-openapi
VERSION="v1alpha1"

serial=`date "+%Y%m%d%H%M%S"`
output_dir="/tmp/${serial}"
mkdir -p ${output_dir}G

# Sensor
go run ${CODEGEN_PKG}/cmd/openapi-gen/openapi-gen.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/pkg/apis/sensor/${VERSION} \
    --output-package github.com/argoproj/argo-events/pkg/apis/sensor/${VERSION} \
    --output-base ${output_dir} \
    $@

# Gateway
go run ${CODEGEN_PKG}/cmd/openapi-gen/openapi-gen.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION} \
    --output-package github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION} \
    --output-base ${output_dir} \
    $@

# EventSource
go run ${CODEGEN_PKG}/cmd/openapi-gen/openapi-gen.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/pkg/apis/eventsources/${VERSION} \
    --output-package github.com/argoproj/argo-events/pkg/apis/eventsources/${VERSION} \
    --output-base ${output_dir} \
    $@
