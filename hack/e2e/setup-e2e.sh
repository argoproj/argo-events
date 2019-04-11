#!/bin/bash

set -e

PROJECT_ROOT=$(cd $(dirname ${BASH_SOURCE})/../..; pwd)

if [[ -z "$E2E_ID" ]]; then
  E2E_ID="argo-events-e2e"
fi

echo "* Set up e2e test $E2E_ID"

kubectl apply -f $PROJECT_ROOT/hack/k8s/manifests/gateway-crd.yaml
kubectl apply -f $PROJECT_ROOT/hack/k8s/manifests/sensor-crd.yaml

echo "* Creating the e2e environment..."
kubectl apply -f - << EOS
apiVersion: v1
kind: Namespace
metadata:
  name: "$E2E_ID"
  labels:
    argo-events-e2e: "$E2E_ID"
EOS

# ls $PROJECT_ROOT/hack/e2e/manifests/* | xargs -I {} sed -e "s|E2E_ID|$E2E_ID|g" {} | kubectl apply -f -
manifests=$(ls $PROJECT_ROOT/hack/e2e/manifests/*)
for m in $manifests; do
  sed -e "s|E2E_ID|$E2E_ID|g" $m | kubectl apply -f -
done

# wait for controllers up
echo "* Wait for controllers startup..."
sleep 3
for i in {1..60}; do
  pods_cnt=$(kubectl get pod -n "$E2E_ID" --no-headers --field-selector "status.phase=Running" 2> /dev/null | wc -l | xargs)
  if [[ $pods_cnt -eq 2 ]]; then
    break
  fi
  if [[ $i -eq 60 ]]; then
    echo "* The controllers didn't start up within the time limit."
    exit 1
  fi
  sleep 1
done
