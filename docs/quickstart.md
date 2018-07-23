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

## 6. Deploy to Minikube
Note: This process is manual right now, but we're working on providing a Helm chart or integrating as a Ksonnet application
```
kubectl create -f hack/k8s/manifests/*
```

## 7. Install minio

```
$ helm init
...
$ helm install stable/minio --name artifacts --set service.type=LoadBalancer
...

$ #Verify that the minio pod, the minio service and minio secret are present
$ kubectl get all -n default -l app=minio

NAME                                   READY     STATUS    RESTARTS   AGE
pod/artifacts-minio-85547b6bd9-bhtt8   1/1       Running   0          21m

NAME                      TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
service/artifacts-minio   ClusterIP   None         <none>        9000/TCP   21m

NAME                              DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/artifacts-minio   1         1         1            1           21m

NAME                                         DESIRED   CURRENT   READY     AGE
replicaset.apps/artifacts-minio-85547b6bd9   1         1         1         21m
```

## 8. Create a bucket in Minio and upload the hello-world.yaml into that bucket.
Download the hello-world.yaml from https://raw.githubusercontent.com/argoproj/argo/master/examples/hello-world.yaml
```
$ kubectl port-forward `kubectl get pod -l app=minio -o name` 9000:9000
```
Open the browser at http://localhost:9000
Create a new bucket called 'workflows'.
Upload the hello-world.yaml into that bucket

## 9. Install Argo
Follow instructions from https://github.com/argoproj/argo/blob/master/demo.md

## 10. Create an example sensor
```
$ kubectl create -f examples/calendar-sensor.yaml
```

Verify that the sensor was created.
```
$ kubectl get sensors -n default
```

Verify that the job associated with this sensor was run
```
$ kubectl get jobs -n default
```

Verify that the Argo workflow was run when the trigger was executed.
```
$ argo list
```

Verify that the sensor was updated correctly
```
$ kubectl get sensors cal-example -n default -o yaml
```

Check the logs of the sensor-controller pod if there are problems.