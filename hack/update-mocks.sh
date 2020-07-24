#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/library.sh
header "updaing mocks"

ensure_mockery

declare -a interfaces=(
# Each line represents a mocked interface,
#
# Format - "relative-dir:interface"
#

"eventbus/driver:Driver" 
)

for i in "${interfaces[@]}"
do
  echo $i
  MOCK_DIR=$(echo $i | awk -F: '{print $1}')
  MOCK_NAME=$(echo $i | awk -F: '{print $2}')
  cd ${REPO_ROOT}/${MOCK_DIR}
  mockery --name ${MOCK_NAME}
done

