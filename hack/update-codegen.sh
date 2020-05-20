#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/library.sh

if [ ! -d "${REPO_ROOT}/vendor" ]; then
  go mod vendor
fi

make_fake_paths

export GOPATH="${FAKE_GOPATH}"
export GO111MODULE="off"

cd "${FAKE_REPOPATH}"

CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${FAKE_REPOPATH}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/sensor github.com/argoproj/argo-events/pkg/apis \
  "sensor:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/gateway github.com/argoproj/argo-events/pkg/apis \
  "gateway:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/eventsources github.com/argoproj/argo-events/pkg/apis \
  "eventsources:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

go run $FAKE_REPOPATH/vendor/k8s.io/gengo/examples/deepcopy-gen/main.go -i github.com/argoproj/argo-events/pkg/apis/common -p github.com/argoproj/argo-events/pkg/apis/common \
   --go-header-file hack/custom-boilerplate.go.txt

# gofmt the tree
find . -name "*.go" -type f -print0 | xargs -0 gofmt -s -w

