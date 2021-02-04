#!/bin/bash
set -eu -o pipefail

source $(dirname $0)/library.sh
ensure_vendor

add_header() {
  cat "$1" | ./hack/auto-gen-msg.sh >tmp
  mv tmp "$1"
}

if [ "$(command -v controller-gen)" = "" ]; then
  go install sigs.k8s.io/controller-tools/cmd/controller-gen
fi

header "Generating CRDs"
controller-gen crd:trivialVersions=true,maxDescLen=0 paths=./pkg/apis/... output:dir=manifests/base/crds

find manifests/base/crds -name 'argoproj.io*.yaml' | while read -r file; do
  echo "Patching ${file}"
  # remove junk fields
  go run ./hack cleancrd "$file"
  add_header "$file"
done

