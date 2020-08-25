#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/library.sh
header "updating swagger"

cd ${REPO_ROOT}
mkdir -p ./dist

VERSION=$1

k8s_swagger="dist/kubernetes.swagger.json"
kubeified_swagger="dist/kubefied.swagger.json"
output="api/openapi-spec/swagger.json"

curl -Ls https://raw.githubusercontent.com/kubernetes/kubernetes/release-1.17/api/openapi-spec/swagger.json -o ${k8s_swagger}

go run ./hack/gen-openapi-spec/main.go ${VERSION} ${k8s_swagger} ${kubeified_swagger}

swagger flatten --with-flatten minimal --with-flatten remove-unused ${kubeified_swagger} -o ${output}

swagger validate ${output}

