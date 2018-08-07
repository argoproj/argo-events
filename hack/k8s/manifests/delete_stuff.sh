#!/bin/sh

alias kargo="kubectl --namespace=argo-events"

kargo delete deployment webhook-example-gateway-deployment

kargo delete deployment gateway-controller

kargo delete service webhook-example-gateway-svc

 kargo delete configmap webhook-example-gateway-transformer-configmap

 kargo delete configmap webhook-gateway-configmap

kargo delete gateway webhook-example-gateway

kargo delete sensor webhook-example

kargo delete deployment sensor-controller

kargo delete job webhook-example

