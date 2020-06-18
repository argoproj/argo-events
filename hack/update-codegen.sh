#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/library.sh
header "running codegen"

ensure_vendor
make_fake_paths

export GOPATH="${FAKE_GOPATH}"
export GO111MODULE="off"

cd "${FAKE_REPOPATH}"

CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${FAKE_REPOPATH}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

#### fix the plural issue of code-generator  ####
for i in `grep '"Endpoints": "Endpoints"' -R vendor/k8s.io/code-generator/ | grep -v EventBus | awk -F\: '{print $1}'`; do
  sed -i "" "s/\"Endpoints\": \"Endpoints\"/\"Endpoints\": \"Endpoints\", \"EventBus\": \"EventBus\"/g" $i
done

subheader "running codegen for sensor"
bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/sensor github.com/argoproj/argo-events/pkg/apis \
  "sensor:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

subheader "running codegen for gateway"
bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/gateway github.com/argoproj/argo-events/pkg/apis \
  "gateway:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

subheader "running codegen for eventsource"
bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/eventsource github.com/argoproj/argo-events/pkg/apis \
  "eventsource:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

subheader "running codegen for eventbus"
bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/eventbus github.com/argoproj/argo-events/pkg/apis \
  "eventbus:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

subheader "running codegen for common"
go run $FAKE_REPOPATH/vendor/k8s.io/gengo/examples/deepcopy-gen/main.go -i github.com/argoproj/argo-events/pkg/apis/common -p github.com/argoproj/argo-events/pkg/apis/common \
   --go-header-file hack/custom-boilerplate.go.txt

# gofmt the tree
subheader "running gofmt"
find . -name "*.go" -type f -print0 | xargs -0 gofmt -s -w

