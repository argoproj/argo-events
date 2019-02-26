# Contributing

## Report a Bug
Open an issue. Please include descriptions of the following:
- Observations
- Expectations
- Steps to reproduce

## Contribute a Bug Fix
- Report the bug first
- Create a pull request for the fix

## Suggest a New Feature
- Create a new issue to start a discussion around new topic. Label the issue as `new-feature`

## Setup your DEV environment
Argo Events is native to Kubernetes so you'll need a running Kubernetes cluster. This guide includes steps for `Minikube` for local development, but if you have another cluster you can ignore the Minikube specific step 3.

### Requirements
- Golang 1.10
- Docker
- dep

### Installation & Setup

#### 1. Get the project
```
go get github.com/argoproj/argo-events
cd $GOPATH/src/github.com/argoproj/argo-events
```

#### 2. Vendor dependencies
```
dep ensure -vendor-only
```

#### 3. Start Minikube and point Docker Client to Minikube's Docker Daemon
```
minikube start
eval $(minikube docker-env)
```

#### 5. Build the project
```
make all
```

Follow [README](README.md#install) to install components.

## Changing Types
If you're making a change to the `pkg/apis`  package, please ensure you re-run the K8 code-generator scripts found in the `/hack` folder. First, ensure you have the `generate-groups.sh` script at the path: `vendor/k8s.io/code-generator/`. Next run the following commands in order:
```
$ make codegen
```
