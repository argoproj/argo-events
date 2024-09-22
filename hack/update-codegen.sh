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

chmod +x ${CODEGEN_PKG}/*.sh

subheader "running codegen"
bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy" \
  github.com/argoproj/argo-events/pkg/client github.com/argoproj/argo-events/pkg/apis \
  "events:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

bash -x ${CODEGEN_PKG}/generate-groups.sh "client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client github.com/argoproj/argo-events/pkg/apis \
  "events:v1alpha1" \
  --plural-exceptions="EventBus:EventBus" \
  --go-header-file hack/custom-boilerplate.go.txt

subheader "running codegen for sensor"
bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/sensor github.com/argoproj/argo-events/pkg/apis \
  "sensor:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

subheader "running codegen for eventsource"
bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/argoproj/argo-events/pkg/client/eventsource github.com/argoproj/argo-events/pkg/apis \
  "eventsource:v1alpha1" \
  --go-header-file hack/custom-boilerplate.go.txt

# gofmt the tree
subheader "running gofmt"
find . -name "*.go" -type f -print0 | xargs -0 gofmt -s -w

