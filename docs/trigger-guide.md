# Trigger

1. [What is a trigger?](#what-is-a-trigger)
2. [What can be a trigger?](#what-can-be-a-trigger)
3. [How to define a trigger?](#how-to-define-a-trigger)
4. [Which triggers are supported out of box?](#which-triggers-are-supported-out-of-box)
5. [How can I add my custom resource as trigger?](#how-can-i-add-my-custom-resource-as-trigger)

## What is a trigger?
Trigger is the resource executed by sensor once the event dependencies are resolved. Any K8s resource can act as a trigger (Custom Resources included). 

## How to define a trigger?
The `resource` field in the trigger object has details of what to execute when the event dependencies have been resolved. 

The framework provides support to fetch trigger resources from different sources.
   * ### Inline
        Inlined artifacts are included directly within the sensor resource and decoded as a string.
   
        [Example](../examples/sensors/artifact.yaml)
   
   * ### S3      
        Argo Events uses the [minio-go](https://github.com/minio/minio-go) client for access to any Amazon S3 compatible object store.
   
        [Example](../examples/sensors/context-filter-webhook.yaml)
    
   * ### File
        Artifacts are defined in a file that is mounted via a [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) within the `sensor-controller` pod.
   
        [Example](../examples/sensors/file-sensor.yaml)
   
   * ### URL
        Artifacts are accessed from web via RESTful API.
   
        [Example](../examples/sensors/url-sensor.yaml)
   
   * ### Configmap
        Artifact stored in Kubernetes configmap are accessed using the key.
   
        [Example](../examples/sensors/trigger-source-configmap.yaml)
   
   * ### Git
        Artifact stored in either public or private Gir repository
   
        [Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/trigger-source-git.yaml)

## What resource types are supported out of box?
- [Argo Workflow](https://github.com/argoproj/argo)
- [Standard K8s resources](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/)
- [Gateway](gateway-protocol.md)
- [Sensor](sensor-protocol.md)

## How can I add my custom resource as trigger?
The set of currently resources supported are implemented in the `store` package. 
Adding support for new resources is as simple as including the type you want to create in the store's `decodeAndUnstructure()` method.
