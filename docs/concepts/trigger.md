# Trigger

## What is a trigger?
Trigger is the resource executed by sensor once the event dependencies are resolved. Any K8s resource can act as a trigger (Custom Resources included). 

## How to define a trigger?
The framework provides support to fetch trigger resources from different sources.

### Inline
Inlined artifacts are included directly within the sensor resource and decoded as a string. [Example](https://github.com/argoproj/argo-events/tree/master/examples/sensors/minio.yaml)
   
### S3      
Argo Events uses the [minio-go](https://github.com/minio/minio-go) client for access to any Amazon S3 compatible object store. [Example](https://github.com/argoproj/argo-events/tree/master/examples/sensors/context-filter-webhook.yaml)
    
### File
Artifacts are defined in a file that is mounted via a [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) within the `sensor-controller` pod. [Example](https://github.com/argoproj/argo-events/tree/master/examples/sensors/trigger-source-file.yaml)
   
### URL
Artifacts are accessed from web via RESTful API. [Example](https://github.com/argoproj/argo-events/tree/master/examples/sensors/url-sensor.yaml)
   
### Configmap
Artifacts stored in Kubernetes configmap are accessed using the key. [Example](https://github.com/argoproj/argo-events/tree/master/examples/sensors/trigger-source-configmap.yaml)
   
### Git
Artifacts stored in either public or private Git repository. [Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/trigger-source-git.yaml)

### Resource
Artifacts defined as generic K8s resource template. This is specially useful if you use tools like Kustomize to generate the sensor spec. 

## What resource types are supported out of box?
- [Argo Workflow](https://github.com/argoproj/argo)
- [Standard K8s resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/)
- [Gateway](gateway.md)
- [Sensor](sensor.md)

## How to trigger standard Kubernetes Resource instead of Argo Workflow?
There could be a case where you may want to trigger a standard Kubernetes resource like Pod, Deployment etc. instead of an Argo Workflow.
The sensor allows you to trigger any K8s resource the same way you would trigger an Argo Workflow.

* The [example](https://github.com/argoproj/argo-events/tree/master/examples/sensors/trigger-standard-k8s-resource.yaml) showcases how you can trigger different standard K8s resources.
  You can find K8s API reference [here](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/).

* To trigger other standard K8s resources, change the group and version in `triggers/resource` accordingly.

## How can I add my custom resource as trigger?
The set of currently supported resources are implemented in the `store` package. 
You need to register your custom resource in order for sensor to be able to  trigger it. Once you register your custom resource, you'll need to rebuild the sensor image. 

Follow these steps,

  1. Go to `store.go` in `store` package.
  2. Import your custom resource api package.
  3. In `init` method, add the scheme to your custom resource api.
  4. Make sure there are no errors.
  5. Rebuild the sensor binary using `make sensor`
  6. To build the image, first change `IMAGE_NAMESPACE` in `Makefile` to your docker registry and then run `make sensor-image`.

