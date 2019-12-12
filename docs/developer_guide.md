# Developer Guide

## Setup your DEV environment
Argo Events is native to Kubernetes so you'll need a running Kubernetes cluster. This guide includes steps for `Minikube` for local development, but if you have another cluster you can ignore the Minikube specific step 3.

### Requirements
- Golang 1.11
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

### Changing Types
If you're making a change to the `pkg/apis`  package, please ensure you re-run the K8 code-generator scripts found in the `/hack` folder. First, ensure you have the `generate-groups.sh` script at the path: `vendor/k8s.io/code-generator/`. Next run the following commands in order:
```
$ make codegen
```


## How to write a custom gateway?
To implement a custom gateway, you need to create a gRPC server and implement the service defined below.
The framework code acts as a gRPC client consuming event stream from gateway server.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/custom-gateway.png?raw=true" alt="Sensor"/>
</p>

<br/>

### Proto Definition
1. The proto file is located [here](https://github.com/argoproj/argo-events/blob/master/gateways/eventing.proto) 

2. If you choose to implement the gateway in `Go`, then you can find generated client stubs [here](https://github.com/argoproj/argo-events/blob/master/gateways/eventing.pb.go)

3. To create stubs in other languages, head over to [gRPC website](https://grpc.io/)

4. Service,

        /**
        * Service for handling event sources.
        */
        service Eventing {
            // StartEventSource starts an event source and returns stream of events.
            rpc StartEventSource(EventSource) returns (stream Event);
            // ValidateEventSource validates an event source.
            rpc ValidateEventSource(EventSource) returns (ValidEventSource);
        }


### Available Environment Variables to Server
 
 | Field                           | Description                                      |
 | ------------------------------- | ------------------------------------------------ |
 | GATEWAY_NAMESPACE               | K8s namespace of the gateway                     |
 | GATEWAY_EVENT_SOURCE_CONFIG_MAP | K8s configmap containing event source            |
 | GATEWAY_NAME                    | name of the gateway                              |
 | GATEWAY_CONTROLLER_INSTANCE_ID  | gateway controller instance id                   |
 | GATEWAY_CONTROLLER_NAME         | gateway controller name                          |
 | GATEWAY_SERVER_PORT             | Port on which the gateway gRPC server should run |
 
