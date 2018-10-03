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

## 2. Deploy Argo Events SA, ClusterRoles, ConfigMap, Sensor Controller and Gateway Controller
Note: Modify the [argo-events-cluster-roles.yaml](../hack/k8s/manifests/argo-events-cluster-roles.yaml) file to use the correct namespace that you wish to deploy the sensor controller and gateway controller.

```
kubectl create namespace argo-events
kubectl apply -f hack/k8s/manifests/argo-events-sa.yaml
kubectl apply -f hack/k8s/manifests/argo-events-cluster-roles.yaml
kubectl apply -f hack/k8s/manifests/sensor-crd.yaml
kubectl apply -f hack/k8s/manifests/gateway-crd.yaml
kubectl apply -f hack/k8s/manifests/sensor-controller-configmap.yaml
kubectl apply -f hack/k8s/manifests/sensor-controller-deployment.yaml
kubectl apply -f hack/k8s/manifests/gateway-controller-configmap.yaml
kubectl apply -f hack/k8s/manifests/gateway-controller-deployment.yaml
```

<b>Note</b> If you want to use a different namespace for deployments, make sure to update namespace references
to your-namespace in above files

## 3. Install Argo
Follow instructions from https://github.com/argoproj/argo/blob/master/demo.md

<b>Note</b>: Make sure to install Argo in `argo-events` namespace

## 4. Create a webhook gateway
```
kubectl apply -f examples/gateways/webhook-gateway-configmap.yaml
kubectl apply -f examples/gateways/webhook.yaml
```

## 5. Create a webhook sensor
```
kubectl apply -f examples/sensors/webhook.yaml
```

## 6. Trigger the webhook & corresponding Argo workflow
Trigger the webhook via sending a http POST request to `/foo` endpoint. You can add different endpoint to 
gateway configuration at run time as well.
Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.
```
export WEBHOOK_SERVICE_URL=$(minikube service --url webhook-gateway-gateway-svc)
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/foo
```

Verify that the Argo workflow was run when the trigger was executed.
```
argo list
```

Verify that the sensor was updated correctly and moved to a "Complete" phase.
```
kubectl get sensor webhook-sensor -o yaml
```

Check the logs of the sensor-controller pod, gateway-controller, associated gateways and sensors if there are problems.

## 7. Next steps
* [Follow tutorial on gateways and sensors](tutorial.md)
* Write your first gateway. Follow the tutorial [Custom Gateways](custom-gateway.md).
