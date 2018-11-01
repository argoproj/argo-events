#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

PROJECT_ROOT=$(cd $(dirname "$0")/.. ; pwd)
CODEGEN_PKG=${PROJECT_ROOT}/vendor/k8s.io/code-generator
VERSION="v1alpha1"

# Sensor
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/pkg/apis/sensor/${VERSION} \
    --output-package github.com/argoproj/argo-events/pkg/apis/sensor/${VERSION} \
    $@

cp ${GOPATH}/src/github.com/argoproj/argo-events/pkg/apis/sensor/${VERSION}/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/sensor.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/sensor.json ${GOPATH}/src/github.com/argoproj/argo-events/pkg/apis/sensor/${VERSION}/


# Gateway
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION} \
    --output-package github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION} \
    $@

cp ${GOPATH}/src/github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION}/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/gateway.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/gateway.json ${GOPATH}/src/github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION}/


# Webhook
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/webhook/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/webhook.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/webhook.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/webhook/

# Artifact
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/artifact/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/artifact.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/artifact.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/artifact/

# Calendar
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/calendar/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/calendar.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/calendar.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/calendar/

# Resource
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/resource/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/resource.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/resource.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/resource/

# File
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/file/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/file.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/file.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/file/

## Streams
# Nats
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/stream/nats/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/nats.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/nats.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/stream/nats/

# Kafka
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/stream/kafka/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/kafka.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/kafka.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/stream/kafka/

# AMQP
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/stream/amqp/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/amqp.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/amqp.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/amqp/

# MQTT
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/stream/mqtt/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/mqtt.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/mqtt.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/core/mqtt/

## Custom
# Storage Grid
cp ${GOPATH}/src/github.com/argoproj/argo-events/gateways/custom/storage-grid/types.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/

go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs k8s.io/kube-openapi/test/integration/argo-events \
    --output-package k8s.io/kube-openapi/test/integration/pkg \
    -p generated \
    -O openapi_generated \
    $@

go run ${GOPATH}/src/k8s.io/kube-openapi/test/integration/builder/main.go ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/storage-grid.json

cp ${GOPATH}/src/k8s.io/kube-openapi/test/integration/argo-events/storage-grid.json ${GOPATH}/src/github.com/argoproj/argo-events/gateways/custom/storage-grid/
