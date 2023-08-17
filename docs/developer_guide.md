# Developer Guide

## Setup your DEV environment

Argo Events is native to Kubernetes so you'll need a running Kubernetes cluster.
This guide includes steps for `Minikube` for local development, but if you have
another cluster you can ignore the Minikube specific step 3.

### Requirements

- Golang 1.20+
- Docker

### Installation & Setup

#### 1. Get the project

```
git clone git@github.com:argoproj/argo-events
cd argo-events
```

#### 2. Start Minikube and point Docker Client to Minikube's Docker Daemon

```
minikube start
eval $(minikube docker-env)
```

#### 3. Build the project

```
make build
```

### Changing Types

If you're making a change to the `pkg/apis` package, please ensure you re-run
following command for code regeneration.

```
make codegen
```
