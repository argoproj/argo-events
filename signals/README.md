# Micro Signals
This package is meant to prove out the feature of leveraging [Micro](https://github.com/micro/go-micro) on [Kubernetes](https://github.com/micro/kubernetes) for building signals as stateless microservices.

## Pre-requisites
- Running Kubernetes cluster >v1.9
- kubectl correctly configured and pointing to your cluster

## Getting started
1. First, modify the [argo-events-cluster-roles.yaml](../hack/k8s/manifests/argo-events-cluster-roles.yaml) file to use the correct namespace that you wish to deploy the sensor controller + signal microservices.
2. Create the `argo-events-sa` service account, associated cluster roles, and cluster role bindings.
```
$ k apply -f hack/k8s/manifests/argo-events-sa.yaml
$ k apply -f hack/k8s/manifests/argo-events-cluster-roles.yaml
```
3. Create Sensor CRD resources
```
$ k apply -f hack/k8s/manifests/sensor-crd.yaml
```
4. Build the latest images
```
$ make all
```
5. Install the sensor controller
```
$ k apply -f hack/k8s/manifests/sensor-controller-configmap.yaml
# k apply -f hack/k8s/manifests/sensor-controller-deployment.yaml
```

## Running a Webhook Signal
1. Create the Webhook Signal Microservice
```
$ k apply -f hack/k8s/manifests/services/webhook.yaml
```
2. Create a webhook sensor
```
$ k apply -f examples/webhook-sensor.yaml
```
3. Trigger the webhook
Using [Postman](https://www.getpostman.com/) or curl, send a POST request to the webhook service.

## Running a NATS Signal
1. Create the NATS Signal Microservice
```
$ k apply -f hack/k8s/manifests/services/stream.yaml
```
2. Create a NATS sensor
```
$ k apply -f examples/nats-sensor.yaml
```
3. Send a NATS message
For `NATS`, you can use `github.com/shogsbro/natscat` you just make sure to expose the NATS service externally as a `LoadBalancer`. 
```
$ go get github.com/shogsbro/natscat
$ cd $GOPATH/src/github.shogsbro/natscat
$ ./natscat -S {insert NATS client endpoint here} -s bucketevents "test"
```
