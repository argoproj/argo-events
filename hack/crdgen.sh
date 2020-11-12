#!/bin/bash
set -eu -o pipefail
echo "0000"
source $(dirname $0)/library.sh
ensure_vendor
echo "1111"
add_header() {
  cat "$1" | ./hack/auto-gen-msg.sh >tmp
  mv tmp "$1"
}
echo "2222"
if [ "$(command -v controller-gen)" = "" ]; then
  go install -mod=vendor ./vendor/sigs.k8s.io/controller-tools/cmd/controller-gen
fi
echo "33333"
header "Generating CRDs"
controller-gen crd:trivialVersions=true,maxDescLen=0 paths=./pkg/apis/... output:dir=manifests/base/crds
echo "4444"
find manifests/base/crds -name 'argoproj.io*.yaml' | while read -r file; do
  echo "Patching ${file}"
  # remove junk fields
  go run ./hack cleancrd "$file"
  add_header "$file"
done

