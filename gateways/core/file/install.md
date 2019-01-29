# File Gateway & Sensor

File system serves as event source for file gateway

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)

## Example event sources definition
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: file-gateway-configmap
data:
  bindir: |- # event source name can be any valid string
    directory: "/bin/"  # directory where file events are watched
    type: CREATE # type of file event
    path: x.txt # file to watch to
```

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/file-gateway-configmap.yaml
```

## Install gateway
Pre-requisite - The file system you want to watch must be mounted in gateway pod and the directory under which a file is to be watched must exist.

1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/file.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway file-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/file.yaml
```
