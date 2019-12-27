# Installation

### Requirements
* Kubernetes cluster >v1.9
* Installed the [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command-line tool >v1.9.0

### Using Helm Chart

Note: This method does not work with Helm 3, only Helm 2.

Make sure you have helm client installed and Tiller server is running. To install helm, follow <a href="https://docs.helm.sh/using_helm/">the link.</a>

1. Add `argoproj` repository

        helm repo add argo https://argoproj.github.io/argo-helm

2. Install `argo-events` chart

        helm install argo/argo-events

### Using kubectl

#### One Command Installation

1. Deploy Argo Events SA, Roles, ConfigMap, Sensor Controller and Gateway Controller
   
        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/installation.yaml

#### Step-by-Step Installation

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

6. Install the event source custom resource definition            

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/event-source-crd.yaml

7. Create the confimap for sensor controller
    
        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-configmap.yaml
    
8. Create the configmap for gateway controller

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-configmap.yaml
    
9. Deploy the sensor controller

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/sensor-controller-deployment.yaml
    
10. Deploy the gateway controller

        kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/gateway-controller-deployment.yaml

## Deploy at cluster level
To deploy Argo-Events controllers at cluster level where the controllers will be 
able to process gateway and sensor objects created in any namespace,

1. Make sure to apply cluster role and binding to the service account,

        kubectl apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/hack/k8s/manifests/argo-events-cluster-roles.yaml

2. Update the configmap for both gateway and sensor and remove the `namespace` key from it.

3. Deploy both gateway and sensor controllers and watch the magic.
