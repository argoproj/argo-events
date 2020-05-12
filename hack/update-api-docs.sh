#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# Setup at https://github.com/ahmetb/gen-crd-api-reference-docs

readonly SCRIPT_ROOT="$(git rev-parse --show-toplevel)"

# Event Source
go run ${SCRIPT_ROOT}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/main.go \
 -config "${SCRIPT_ROOT}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1" \
 -out-file "${SCRIPT_ROOT}/api/event-source.html" \
 -template-dir "${SCRIPT_ROOT}/hack/api-docs-template"

# Gateway
go run ${SCRIPT_ROOT}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/main.go \
 -config "${SCRIPT_ROOT}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1" \
 -out-file "${SCRIPT_ROOT}/api/gateway.html" \
 -template-dir "${SCRIPT_ROOT}/hack/api-docs-template"

# Sensor
go run ${SCRIPT_ROOT}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/main.go \
 -config "${SCRIPT_ROOT}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1" \
 -out-file "${SCRIPT_ROOT}/api/sensor.html" \
 -template-dir "${SCRIPT_ROOT}/hack/api-docs-template"

# Setup at https://pandoc.org/installing.html

pandoc --from markdown --to gfm ${SCRIPT_ROOT}/api/event-source.html > ${SCRIPT_ROOT}/api/event-source.md
pandoc --from markdown --to gfm ${SCRIPT_ROOT}/api/gateway.html > ${SCRIPT_ROOT}/api/gateway.md
pandoc --from markdown --to gfm ${SCRIPT_ROOT}/api/sensor.html > ${SCRIPT_ROOT}/api/sensor.md
