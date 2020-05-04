#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

serial=`date "+%Y%m%d%H%M%S"`
output_dir="/tmp/${serial}"
mkdir -p ${output_dir}

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/sensor github.com/argoproj/argo-events/pkg/apis \
  "sensor:v1alpha1" \
  --go-header-file $SCRIPT_ROOT/hack/custom-boilerplate.go.txt --output-base ${output_dir}

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/gateway github.com/argoproj/argo-events/pkg/apis \
  "gateway:v1alpha1" \
  --go-header-file $SCRIPT_ROOT/hack/custom-boilerplate.go.txt --output-base ${output_dir}

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/eventsources github.com/argoproj/argo-events/pkg/apis \
  "eventsources:v1alpha1" \
  --go-header-file $SCRIPT_ROOT/hack/custom-boilerplate.go.txt --output-base ${output_dir}

go run $SCRIPT_ROOT/vendor/k8s.io/gengo/examples/deepcopy-gen/main.go -i github.com/argoproj/argo-events/pkg/apis/common -p github.com/argoproj/argo-events/pkg/apis/common \
   --output-base ${output_dir} \
   --go-header-file $SCRIPT_ROOT/hack/custom-boilerplate.go.txt

cp -r ${output_dir}/github.com/argoproj/argo-events/* ${SCRIPT_ROOT}/
rm -rf ${output_dir}

