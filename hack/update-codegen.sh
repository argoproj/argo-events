#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..
CODEGEN_PKG="$SCRIPT_ROOT/vendor/k8s.io/code-generator"

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
				  github.com/argoproj/argo-events/pkg/client/sensor github.com/argoproj/argo-events/pkg/apis \
					  "sensor:v1alpha1" \
						  --go-header-file $SCRIPT_ROOT/hack/custom-boilerplate.go.txt

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
				  github.com/argoproj/argo-events/pkg/client/gateway github.com/argoproj/argo-events/pkg/apis \
					  "gateway:v1alpha1" \
						  --go-header-file $SCRIPT_ROOT/hack/custom-boilerplate.go.txt

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
				  github.com/argoproj/argo-events/pkg/client/eventsources github.com/argoproj/argo-events/pkg/apis \
					  "eventsources:v1alpha1" \
						  --go-header-file $SCRIPT_ROOT/hack/custom-boilerplate.go.txt

go run $SCRIPT_ROOT/vendor/k8s.io/gengo/examples/deepcopy-gen/main.go -i github.com/argoproj/argo-events/pkg/apis/common -p github.com/argoproj/argo-events/pkg/apis/common --go-header-file $SCRIPT_ROOT/vendor/k8s.io/gengo/boilerplate/boilerplate.go.txt
