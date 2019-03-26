#!/bin/bash

set -e

kubectl delete ns -l argo-events-e2e
