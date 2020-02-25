#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# Setup at https://github.com/ahmetb/gen-crd-api-reference-docs

# Event Source
${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/gen-crd-api-reference-docs \
 -config "${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1" \
 -out-file "${GOPATH}/src/github.com/argoproj/argo-events/api/event-source.html" \
 -template-dir "${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/template"

# Gateway
${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/gen-crd-api-reference-docs \
 -config "${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1" \
 -out-file "${GOPATH}/src/github.com/argoproj/argo-events/api/gateway.html" \
 -template-dir "${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/template"

# Sensor
${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/gen-crd-api-reference-docs \
 -config "${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1" \
 -out-file "${GOPATH}/src/github.com/argoproj/argo-events/api/sensor.html" \
 -template-dir "${GOPATH}/src/github.com/ahmetb/gen-crd-api-reference-docs/template"

# Setup at https://pandoc.org/installing.html

pandoc --from markdown --to gfm ${GOPATH}/src/github.com/argoproj/argo-events/api/event-source.html > ${GOPATH}/src/github.com/argoproj/argo-events/api/event-source.md
pandoc --from markdown --to gfm ${GOPATH}/src/github.com/argoproj/argo-events/api/gateway.html > ${GOPATH}/src/github.com/argoproj/argo-events/api/gateway.md
pandoc --from markdown --to gfm ${GOPATH}/src/github.com/argoproj/argo-events/api/sensor.html > ${GOPATH}/src/github.com/argoproj/argo-events/api/sensor.md
