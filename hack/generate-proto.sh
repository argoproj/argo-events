#!/bin/bash

# This script auto-generates protobuf related files. It is intended to be run manually when either
# API types are added/modified, or server gRPC calls are added. The generated files should then
# be checked into source control.

set -x
set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/..; pwd)
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${PROJECT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}
PATH="${PROJECT_ROOT}/dist:${PATH}"


# protbuf tooling required to build .proto files from go annotations from k8s-like api types
go build -i -o dist/go-to-protobuf ./vendor/k8s.io/code-generator/cmd/go-to-protobuf
go build -i -o dist/protoc-gen-gogo ./vendor/k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo

# Generate pkg/apis/<group>/<apiversion>/(generated.proto,generated.pb.go)
# NOTE: any dependencies of our types to the k8s.io apimachinery types should be added to the
# --apimachinery-packages= option so that go-to-protobuf can locate the types, but prefixed with a
# '-' so that go-to-protobuf will not generate .proto files for it.
PACKAGES=(
    github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1
)
APIMACHINERY_PKGS=(
    +k8s.io/apimachinery/pkg/util/intstr
    +k8s.io/apimachinery/pkg/api/resource
    +k8s.io/apimachinery/pkg/runtime/schema
    +k8s.io/apimachinery/pkg/runtime
    k8s.io/apimachinery/pkg/apis/meta/v1
    k8s.io/api/core/v1
)
go-to-protobuf \
    --logtostderr \
    --go-header-file=${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --packages=$(IFS=, ; echo "${PACKAGES[*]}") \
    --apimachinery-packages=$(IFS=, ; echo "${APIMACHINERY_PKGS[*]}") \
    --proto-import=./vendor

# Either protoc-gen-go, protoc-gen-gofast, or protoc-gen-gogofast can be used to build
# server/*/<service>.pb.go from .proto files. golang/protobuf and gogo/protobuf can be used
# interchangeably. The difference in the options are:
# 1. protoc-gen-go - official golang/protobuf
go build -i -o dist/protoc-gen-go ./vendor/github.com/golang/protobuf/protoc-gen-go
GOPROTOBINARY=go
# 2. protoc-gen-gofast - fork of golang golang/protobuf. Faster code generation
#go build -i -o dist/protoc-gen-gofast ./vendor/github.com/gogo/protobuf/protoc-gen-gofast
#GOPROTOBINARY=gofast
# 3. protoc-gen-gogofast - faster code generation and gogo extensions and flexibility in controlling
# the generated go code (e.g. customizing field names, nullable fields)
#go build -i -o dist/protoc-gen-gogofast ./vendor/github.com/gogo/protobuf/protoc-gen-gogofast
#GOPROTOBINARY=gogofast

# Generate job/<service>/(<service>.pb.go)
PROTO_FILES=$(find $PROJECT_ROOT \( -name "*.proto" -and -path '*/shared/*' \))
for i in ${PROTO_FILES}; do
    protoc \
        -I${PROJECT_ROOT} \
        -I/usr/local/include \
        -I./vendor \
        -I$GOPATH/src \
        --${GOPROTOBINARY}_out=plugins=grpc:$GOPATH/src \
        $i
done