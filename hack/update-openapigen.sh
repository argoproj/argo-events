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

# Gateway
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION} \
    --output-package github.com/argoproj/argo-events/pkg/apis/gateway/${VERSION} \
    $@

# Webhook
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/webhook \
    --output-package github.com/argoproj/argo-events/gateways/core/webhook \
    $@

# Artifact
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/artifact \
    --output-package github.com/argoproj/argo-events/gateways/core/artifact \
    $@

# Calendar
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/calendar/ \
    --output-package github.com/argoproj/argo-events/gateways/core/calendar/ \
    $@

# Resource
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/resource/ \
    --output-package github.com/argoproj/argo-events/gateways/core/resource/ \
    $@

# File
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/file/ \
    --output-package github.com/argoproj/argo-events/gateways/core/file/ \
    $@

## Streams
# Nats
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/stream/nats \
    --output-package github.com/argoproj/argo-events/gateways/core/stream/nats \
    $@

# Kafka
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/stream/kafka \
    --output-package github.com/argoproj/argo-events/gateways/core/stream/kafka \
    $@

# AMQP
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/stream/amqp \
    --output-package github.com/argoproj/argo-events/gateways/core/stream/amqp \
    $@

# MQTT
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/core/stream/mqtt \
    --output-package github.com/argoproj/argo-events/gateways/core/stream/mqtt \
    $@

## Custom
# Storage Grid
go run ${CODEGEN_PKG}/cmd/openapi-gen/main.go \
    --go-header-file ${PROJECT_ROOT}/hack/custom-boilerplate.go.txt \
    --input-dirs github.com/argoproj/argo-events/gateways/custom/storagegrid \
    --output-package github.com/argoproj/argo-events/gateways/custom/storagegrid \
    $@
