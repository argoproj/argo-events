# Artifact Gateway & Sensor

Minio bucket notifications acts as event sources for artifact gateway. To setup minio, follow  https://www.minio.io/kubernetes.html

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)

## Example event sources definition
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-gateway-configmap
data:  
  input: |-
    bucket: 
      name: input # name of the bucket we want to listen to
    endpoint: minio-service.argo-events:9000 # minio service endpoint
    event: s3:ObjectCreated:Put # type of event
    filter: # filter on object name if any
      prefix: ""
      suffix: ""
    insecure: true # type of minio server deployment
    accessKey: 
      key: accesskey # key within below k8 secret whose corresponding value is name of the accessKey
      name: artifacts-minio # k8 secret name that holds minio creds
    secretKey:
      key: secretkey # key within below k8 secret whose corresponding value is name of the secretKey
      name: artifacts-minio # k8 secret name that holds minio creds
``` 

```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/artifact-gateway-configmap.yaml
```

## Install Gateway
Pre-requisite - create necessary buckets in Minio.
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/artifact-http.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway artifact-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/artifact.yaml
```
