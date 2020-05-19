#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# Setup at https://github.com/ahmetb/gen-crd-api-reference-docs

source $(dirname $0)/library.sh

if [ ! -d "${REPO_ROOT}/vendor" ]; then
  export GO111MODULE="on"
  go mod vendor
fi

make_fake_paths

export GOPATH="${FAKE_GOPATH}"
export GO111MODULE="off"

cd "${FAKE_REPOPATH}"

# Event Source
go run ${FAKE_REPOPATH}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/main.go \
 -config "${FAKE_REPOPATH}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1" \
 -out-file "${FAKE_REPOPATH}/api/event-source.html" \
 -template-dir "${FAKE_REPOPATH}/hack/api-docs-template"

# Gateway
go run ${FAKE_REPOPATH}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/main.go \
 -config "${FAKE_REPOPATH}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1" \
 -out-file "${FAKE_REPOPATH}/api/gateway.html" \
 -template-dir "${FAKE_REPOPATH}/hack/api-docs-template"

# Sensor
go run ${FAKE_REPOPATH}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/main.go \
 -config "${FAKE_REPOPATH}/vendor/github.com/ahmetb/gen-crd-api-reference-docs/example-config.json" \
 -api-dir "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1" \
 -out-file "${FAKE_REPOPATH}/api/sensor.html" \
 -template-dir "${FAKE_REPOPATH}/hack/api-docs-template"

# Setup at https://pandoc.org/installing.html

pandoc --from markdown --to gfm ${FAKE_REPOPATH}/api/event-source.html > ${FAKE_REPOPATH}/api/event-source.md
pandoc --from markdown --to gfm ${FAKE_REPOPATH}/api/gateway.html > ${FAKE_REPOPATH}/api/gateway.md
pandoc --from markdown --to gfm ${FAKE_REPOPATH}/api/sensor.html > ${FAKE_REPOPATH}/api/sensor.md

