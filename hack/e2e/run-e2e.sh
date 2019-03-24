#!/bin/bash

set -e

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/../..; pwd)

export E2E_ID=${E2E_ID:-argo-events-e2e-$(date +%s)}

$PROJECT_ROOT/hack/e2e/setup-e2e.sh

function cleanup {
  if [[ -z "$KEEP_NAMESPACE" ]]; then
    echo "* Cleaning up the e2e environment..."
    kubectl delete ns $E2E_ID
  else
    echo "* Skip e2e environment cleanup for $E2E_ID."
  fi
}
trap cleanup EXIT

echo "* Run e2e tests."
go test -v ./test/e2e/...
