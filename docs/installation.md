# Installation


### Requirements
* Kubernetes cluster >v1.9
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.9.0

### Helm Chart

Make sure you have helm client installed and Tiller server is running. To install helm, follow <a href="https://docs.helm.sh/using_helm/">the link.</a>

1. Add `argoproj` repository

        helm repo add argo https://argoproj.github.io/argo-helm

2. Install `argo-events` chart

        helm install argo/argo-events
   

### Using kubectl
* Deploy Argo Events SA, Roles, ConfigMap, Sensor Controller and Gateway Controller

1. Create the namespace

        kubectl create namespace argo-events

2. Create the service account
              
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-sa.yaml
  
3. Create the cluster roles

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-cluster-roles.yaml
        
4. Install the sensor custom resource definition

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-crd.yaml
    
5. Install the gateway custom resource definition

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-crd.yaml
            
6. Create the confimap for sensor controller
    
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-configmap.yaml
    
7. Create the configmap for gateway controller

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-configmap.yaml
    
8. Deploy the sensor controller

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-deployment.yaml
    
9. Deploy the gateway controller

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-deployment.yaml
