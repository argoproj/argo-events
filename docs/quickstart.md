# Getting Started - Quickstart
This is a guide to getting started with Argo Events using Minikube.

## Requirements
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.9.0
* Have a [kubeconfig](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) file (default location is `~/.kube/config`).
* Installed Minikube >v0.26.1
* Installed Go >1.9 and properly setup the [GOPATH](https://golang.org/doc/install)
* Installed [dep](https://golang.github.io/dep/docs/installation.html), Go's dependency tool

## 1. Checkout project's master branch
```
$ git clone git@github.com:argoproj/argo-events.git
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

## 5. Build the project & Docker images
```
$ cd go/src/github.com/argoproj/argo-events
$ make all
```

## 6. Deploy Argo Events SA, ClusterRoles, ConfigMap, and Sensor Controller
Note 1: This process is manual right now, but we're working on providing a Helm chart or integrating as a Ksonnet application
Note 2: Modify the [argo-events-cluster-roles.yaml](../hack/k8s/manifests/argo-events-cluster-roles.yaml) file to use the correct namespace that you wish to deploy the sensor controller + signal microservices.
```
$ k apply -f hack/k8s/manifests/argo-events-sa.yaml
$ k apply -f hack/k8s/manifests/argo-events-cluster-roles.yaml
$ k apply -f hack/k8s/manifests/sensor-crd.yaml
$ k apply -f hack/k8s/manifests/sensor-controller-configmap.yaml
$ k apply -f hack/k8s/manifests/sensor-controller-deployment.yaml
```

## 7. Deploy Argo Events Webhook Signal Microservice
Note: You will need to separately deploy the various signal services that you wish to support.
```
$ k apply -f hack/k8s/manifests/services/webhook.yaml
```

## 8. Install Argo
Follow instructions from https://github.com/argoproj/argo/blob/master/demo.md

## 9. Create a webhook sensor
```
$ k apply -f examples/webhook-with-resource-param.yaml
```

Verify that the sensor was created.
```
$ kubectl get sensors -n default
```

Verify that the signal microservice is listening for signals and the sensor is active.
```
$ kubectl logs signal-webhook-xxx -f
$ kubectl get sensor webhook-with-resource-param -n default -o yaml
```

## 10. Trigger the webhook & corresponding Argo workflow
Trigger the webhook via sending a POST with a JSON with a "message" key and value. 
Ensure that you set the header "Content-Type" to "application/json" or this event will be ignored.
```
$ curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $(minikube service --url webhook)
```

Verify that the Argo workflow was run when the trigger was executed.
```
$ argo list
```

Verify that the sensor was updated correctly and moved to a "Complete" phase
```
$ kubectl get sensor webhook-with-resource-param -n default -o yaml
```

Check the logs of the Argo workflow pod for the message you posted.
```
$ k logs arguments-via-webhook-event main
```

Check the logs of the sensor-controller pod or the associated signal microservice if there are problems.