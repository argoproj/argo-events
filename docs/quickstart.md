# Getting Started - Quickstart
This is a guide to getting started with using Axis using Minikube.

## Requirements
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.9.0
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).
* Installed Minikube >v0.26.1
* Installed Go >1.9 and properly setup the [GOPATH](https://golang.org/doc/install)
* Installed [dep](https://golang.github.io/dep/docs/installation.html), Go's dependency tool

## 1. Checkout project's master branch
```
$ git clone git@github.com:/axis.git
```

## 2. Install vendor dependencies
```
$ dep ensure -vendor-only
```

## 3. Start Minikube
```
$ minikube start
```

## 4. Point Docker Client to Minikube's Docker Daemon
```
$ eval $(minikube docker-env)
```

## 5. Build the Axis project & Docker images
```
$ cd go/src/github.com/blackrock/axis
$ make all
```

## 6. Deploy to Minikube
Note: This process is manual right now, but we're working on providing a Helm chart or integrating as a Ksonnet application
```
kubectl create -f hack/k8s/manifests/*
```

## 7. Creating a sensor
See the `examples/` directory for a list of sample `Sensors`. Once the `sensor-controller` is deployed, creating a sensor is easy as:
```
kubectl create -f examples/calendar-sensor.yaml
```
