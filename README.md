# Argo Events - The Event-Based Dependency Manager for Kubernetes

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj/argo-events)](https://goreportcard.com/report/github.com/argoproj/argo-events)
[![slack](https://img.shields.io/badge/slack-argoproj-brightgreen.svg?logo=slack)](https://argoproj.github.io/community/join-slack)

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/argo-events-logo.png?raw=true" alt="Logo"/>
</p>

## What is Argo Events?
**Argo Events** is an event-based dependency manager for Kubernetes which helps you define multiple dependencies from a variety of event sources like webhook, s3, schedules, streams etc.
and trigger Kubernetes objects after successful event dependencies resolution.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/argo-events-top-level.png?raw=true" alt="High Level Overview"/>
</p>

<br/>

## Features 
* Manage dependencies from a variety of event sources.
* Ability to customize business-level constraint logic for event dependencies resolution.
* Manage everything from simple, linear, real-time dependencies to complex, multi-source, batch job dependencies.
* Ability to extends framework to add your own event source listener.
* Define arbitrary boolean logic to resolve event dependencies.
* CloudEvents compliant.
* Ability to manage event sources at runtime.

## Core Concepts
The framework is made up of two components: 

 1. **`Gateway`** which is implemented as a Kubernetes-native Custom Resource Definition processes events from event source.

 2. **`Sensor`** which is implemented as a Kubernetes-native Custom Resource Definition defines a set of event dependencies and triggers K8s resources.

## Install

* ### Requirements
  * Kubernetes cluster >v1.9
  * Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.9.0

* ### Helm Chart

    Make sure you have helm client installed and Tiller server is running. To install helm, follow https://docs.helm.sh/using_helm/

    1. Add `argoproj` repository

    ```bash
    helm repo add argo https://argoproj.github.io/argo-helm
    ```

    2. Install `argo-events` chart
    
    ```bash
    helm install argo/argo-events
    ```   

* ### Using kubectl
  * Deploy Argo Events SA, Roles, ConfigMap, Sensor Controller and Gateway Controller
  
    ```
    kubectl create namespace argo-events
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-sa.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-cluster-roles.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-crd.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-crd.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-configmap.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-deployment.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-configmap.yaml
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-deployment.yaml
    ```

**Note**: If you have already deployed the argo workflow controller in another namespace
and the controller is namespace scoped, make sure to deploy a new controller in `argo-events` namespace.  

## Get Started
Lets deploy a webhook gateway and sensor,

 * First, we need to setup event sources for gateway to listen. The event sources for any gateway are managed using K8s configmap.
   
   ```bash
   kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook-gateway-configmap.yaml 
   ```
   
 * Create webhook gateway, 
 
   ```bash
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook-http.yaml
   ```
    
   After running above command, gateway controller will create corresponding gateway pod and a LoadBalancing service.
 
 * Create webhook sensor,
    
    ```bash
    kubectl apply -n argo-events https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook-http.yaml
    ```
    
   Once sensor resource is created, sensor controller will create corresponding sensor pod and a ClusterIP service. 
    
 * Once the gateway and sensor pods are running, trigger the webhook via a http POST request to `/foo` endpoint.
   Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.
   ```
    export WEBHOOK_SERVICE_URL=$(minikube service -n argo-events --url <gateway_service_name>)
    echo $WEBHOOK_SERVICE_URL
    curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/foo
   ```
 
   <b>Note</b>: 
     * If you are facing an issue getting service url by running `minikube service -n argo-events --url <gateway_service_name>`, you can use `kubectl port-forward`
     * Open another terminal window and enter `kubectl port-forward -n argo-events <name_of_the_webhook_gateway_pod> 9003:<port_on_which_gateway_server_is_running>`
     * You can now use `localhost:9003` to query webhook gateway
   
   Verify that the Argo workflow was run when the trigger was executed.
   ```
   argo list -n argo-events
   ```

 * More examples can be found at [examples](./examples)

## Further Reading
1. [Gateway](docs/gateway-guide.md)
2. [Sensor](docs/sensor-guide.md)
3. [Trigger](docs/trigger-guide.md)
4. [Communication between gateway and sensor](docs/communication.md)
5. [Controllers](docs/controllers-guide.md)

## Contribute
[Contributions](https://github.com/argoproj/argo-events/issues) are more than welcome, if you are interested please take a look at our [Contributing Guidelines](./CONTRIBUTING.md).

## License
Apache License Version 2.0, see [LICENSE](./LICENSE)
