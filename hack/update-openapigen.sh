#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/library.sh
header "updating open-apis"

ensure_vendor

CODEGEN_PKG=${REPO_ROOT}/vendor/k8s.io/kube-openapi
VERSION="v1alpha1"

go run ${CODEGEN_PKG}/cmd/openapi-gen/openapi-gen.go \
    --go-header-file ${REPO_ROOT}/hack/custom-boilerplate.go.txt \
    --output-dir pkg/apis/events/openapi \
    --output-pkg openapi \
    --output-file openapi_generated.go \
    github.com/argoproj/argo-events/pkg/apis/events/${VERSION}

