# Installation
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
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-sa.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-cluster-roles.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-crd.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-crd.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-configmap.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-deployment.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-configmap.yaml
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-deployment.yaml
```

<b>Note</b> If you want to use a different namespace for deployments, make sure to update namespace references
to your-namespace in above files

## 3. Install Argo
Follow instructions from https://github.com/argoproj/argo/blob/master/demo.md

<b>Note</b>: Make sure to install Argo in `argo-events` namespace

## 4. Create a webhook gateway
```
kubectl apply -n argo-events -f examples/gateways/webhook-gateway-configmap.yaml
kubectl apply -n argo-events -f examples/gateways/webhook.yaml
```

## 5. Create a webhook sensor
```
kubectl apply -n argo-events -f examples/sensors/webhook.yaml
```

## 6. Trigger the webhook & corresponding Argo workflow
Trigger the webhook via sending a http POST request to `/foo` endpoint. You can add different endpoint to 
gateway configuration at run time as well.
Note: the `WEBHOOK_SERVICE_URL` will differ based on the Kubernetes cluster.
```
export WEBHOOK_SERVICE_URL=$(minikube service -n argo-events --url webhook-gateway-gateway-svc)
echo $WEBHOOK_SERVICE_URL
curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST $WEBHOOK_SERVICE_URL/foo
```

<b>Note</b>: 
   * If you are facing an issue getting service url by running `minikube service -n argo-events --url webhook-gateway-gateway-svc`, you can use `kubectl port-forward`
   * Open another terminal window and enter `kubectl port-forward -n argo-events <name_of_the_webhook_gateway_pod> 9003:<port_on_which_gateway_server_is_running>`
   * You can now user `localhost:9003` to query webhook gateway
  

Verify that the Argo workflow was run when the trigger was executed.
```
argo list -n argo-events
```

Verify that the sensor was updated correctly and moved to a "Complete" phase.
```
kubectl get -n argo-events sensor webhook-sensor -o yaml
```

Check the logs of the sensor-controller pod, gateway-controller, associated gateways and sensors if there are problems.

## 7. Next steps
* [Follow tutorial on gateways and sensors](tutorial.md)
* Write your first gateway. Follow the tutorial [Custom Gateways](custom-gateway.md).
