#!/bin/bash
set -eux -o pipefail

for dir in $(find examples -mindepth 2 -type d | sort); do
  kubectl create --dry-run=server -f $dir
done
