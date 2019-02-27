# Resource Gateway & Sensor

K8s resources act as event sources for Resource gateway

1. [Example event sources definition](#example-event-sources-definition)
2. [Install gateway](#install-gateway)
3. [Install sensor](#install-sensor)
4. [Trigger Workflow](#trigger-workflow)

## Example event sources definition
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: resource-gateway-configmap
data:
  successWorkflow: |-
    namespace: argo-events # namespace where wf is deloyed
    group: "argoproj.io" # wf group
    version: "v1alpha1" # wf version
    kind: "Workflow" # object kind
    filter: # filters can be applied on labels, annotations, creation time and name 
      labels:
        workflows.argoproj.io/phase: Succeeded
        name: "my-workflow"
  failureWorkflow: |-
    namespace: argo-events
    group: "argoproj.io"
    version: "v1alpha1"
    kind: "Workflow"
    filter:
      prefix: scripts-bash
      labels:
        workflows.argoproj.io/phase: Failed
```

Create gateway event sources

```yaml
kubectl -n argo-events create -f  https://github.com/argoproj/argo-events/blob/master/examples/gateways/resource-gateway-configmap.yaml
```

## Install gateway
1. **Create gateway**

    ```yaml
    kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/gateways/resource.yaml
    ```

2. **Check the status of the gateway**
    
    ```yaml
    kubectl -n argo-events describe gateway resource-gateway
    ```
    
   Make sure the gateway is in active state and all the event sources are in running state.
   
## Install Sensor
```yaml
kubectl -n argo-events create -f https://github.com/argoproj/argo-events/blob/master/examples/sensors/resource.yaml
```

## Trigger workflow
Create an argo workflow with name `my-workflow`. As soon as `my-workflow` succeeds, a new workflow will be triggered.
