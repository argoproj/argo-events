#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/library.sh
header "generating proto files"

ensure_vendor
make_fake_paths

export GOPATH="${FAKE_GOPATH}"
export GO111MODULE="off"

cd "${FAKE_REPOPATH}"

go install ./vendor/k8s.io/code-generator/cmd/go-to-protobuf

${GOPATH}/bin/go-to-protobuf \
        --go-header-file=./hack/custom-boilerplate.go.txt \
        --packages=github.com/argoproj/argo-events/pkg/apis/common \
        --apimachinery-packages=+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1 \
        --proto-import ./vendor

${GOPATH}/bin/go-to-protobuf \
        --go-header-file=./hack/custom-boilerplate.go.txt \
        --packages=github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1,github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1,github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1 \
        --apimachinery-packages=github.com/argoproj/argo-events/pkg/apis/common,+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1 \
        --proto-import ./vendor

