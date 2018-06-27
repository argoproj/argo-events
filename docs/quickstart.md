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
$ helm install stable/minio
...

$ #Verify that the minio pod, the minio service and minio secret are present
$ kubectl get all -n default -l app=minio

NAME                                        READY     STATUS    RESTARTS   AGE
pod/laughing-llama-minio-6586bf655c-c5wx2   1/1       Running   0          15h

NAME                           TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
service/laughing-llama-minio   ClusterIP   None         <none>        9000/TCP   15h

NAME                                   DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/laughing-llama-minio   1         1         1            1           15h

NAME                                              DESIRED   CURRENT   READY     AGE
replicaset.apps/laughing-llama-minio-6586bf655c   1         1         1         15h
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
The example sensor below expects a secret called `artifacts-minio` in the default namespace. Change the name from `artifacts-minio` to the actual name of the secret created by minio. In the example below, minion created a secret called `laughing-llama-minio`.
```
$ kubectl get secret -l app=minio -n default
laughing-llama-minio

$ #Replace `artifacts-minio` with `laughing-llama-minio` in the yaml. Then execute the following command.
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