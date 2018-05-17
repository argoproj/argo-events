#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT=$(cd $(dirname "$0")/.. ; pwd)
CODEGEN_PKG=${PROJECT_ROOT}/vendor/k8s.io/code-generator
VERSION="v1alpha1"

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
  --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
  --input-dirs github.com/blackrock/axis/pkg/apis/sensor/${VERSION} \
  --output-package github.com/blackrock/axis/pkg/apis/sensor/${VERSION} \
  $@
