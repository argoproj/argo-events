# Argo Events

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/argo-events-logo.png?raw=true" alt="Logo"/>
</p>

## What is Argo Events?
**Argo Events** is an event-based dependency manager for Kubernetes which helps you define multiple dependencies from a variety of event sources like webhook, s3, schedules, streams etc.
and trigger Kubernetes objects after successful event dependencies resolution.

<br/>
<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/master/docs/assets/argo-events-top-level.png?raw=true" alt="High Level Overview"/>
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

 1. **Gateway** which is implemented as a Kubernetes-native Custom Resource Definition processes events from event source.

 2. **Sensor** which is implemented as a Kubernetes-native Custom Resource Definition defines a set of event dependencies and triggers K8s resources.

 3. **Event Source** is a configmap that contains configurations which is interpreted by gateway as source for events producing entity. 
 
## Get Started
Lets deploy a webhook gateway and sensor,

 * First, we need to setup event sources for gateway to listen. The event sources for any gateway are managed using K8s configmap.
   
   ```bash
   kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/event-sources/webhook.yaml 
   ```
   
 * Create webhook gateway, 
 
   ```bash
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/webhook.yaml
   ```
    
   After running above command, gateway controller will create corresponding gateway pod and a LoadBalancing service.
 
 * Create webhook sensor,
    
    ```bash
    kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/webhook.yaml
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
