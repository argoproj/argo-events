# Getting Started - Quickstart
This is a guide to getting started with Argo Events.

## Requirements
* Kubernetes cluster >v1.9
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.9.0
* Installed Go >1.9 and properly setup the [GOPATH](https://golang.org/doc/install) environment variable
* Installed [dep](https://golang.github.io/dep/docs/installation.html), Go's dependency tool

## 1. Get the project
```
go get github.com/argoproj/argo-events
cd $GOPATH/src/github.com/argoproj/argo-events
```

## 2. Deploy Argo Events SA, ClusterRoles, ConfigMap, and Sensor Controller
Note 1: This process is manual right now, but we're working on providing a Helm chart or integrating as a Ksonnet application.
Note 2: Modify the [argo-events-cluster-roles.yaml](../hack/k8s/manifests/argo-events-cluster-roles.yaml) file to use the correct namespace that you wish to deploy the sensor controller + signal microservices.
```
kubectl apply -f hack/k8s/manifests/argo-events-sa.yaml
kubectl apply -f hack/k8s/manifests/argo-events-cluster-roles.yaml
kubectl apply -f hack/k8s/manifests/sensor-crd.yaml
kubectl apply -f hack/k8s/manifests/sensor-controller-configmap.yaml
kubectl apply -f hack/k8s/manifests/sensor-controller-deployment.yaml
```

## 3. Deploy Argo Events Webhook Signal Microservice
Note: In order to have a useful cluster for Argo Events, you will need to separately deploy the various signal services that you wish to support. This command installs the webhook signal service.
```
kubectl apply -f hack/k8s/manifests/services/webhook.yaml
```

## 4. Install Argo
Follow instructions from https://github.com/argoproj/argo/blob/master/demo.md

## 5. Create a webhook sensor
```
kubectl apply -f examples/webhook-with-resource-param.yaml
```

Verify that the sensor was created.
```
kubectl get sensors -n default
```

Verify that the signal microservice is listening for signals and the sensor is active.
```
kubectl logs signal-webhook-xxx -f
kubectl get sensor webhook-with-resource-param -n default -o yaml
```

## 6. Trigger the webhook & corresponding Argo workflow
Trigger the webhook via sending a POST with a JSON with a "message" key and value. 
Ensure that you set the header "Content-Type" to "application/json" or this event will be ignored.
Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.
```
export WEBHOOK_SERVICE_URL=$(minikube service --url webhook)
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/hello
```

Verify that the Argo workflow was run when the trigger was executed.
```
argo list
```

Verify that the sensor was updated correctly and moved to a "Complete" phase
```
kubectl get sensor webhook-with-resource-param -n default -o yaml
```

Check the logs of the Argo workflow pod for the message you posted.
```
kubectl logs arguments-via-webhook-event main
```

Check the logs of the sensor-controller pod or the associated signal microservice if there are problems.