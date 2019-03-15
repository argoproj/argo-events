<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/assets/k8s.png?raw=true" alt="K8s"/>
</p>

<br/>


# Resource

Resource gateway listens to updates on **any** Kubernetes resource.

You consider using resource gateway as the watchdog in your cluster, basically monitoring various updated to 
installed kubernetes resources. 

## How to define an event source in confimap?
You can construct an entry in configmap using following fields,

```go
// Namespace where resource is deployed
Namespace string `json:"namespace"`

// Filter is applied on the metadata of the resource
Filter *ResourceFilter `json:"filter,omitempty"`

// Version of the source
Version string `json:"version"`

// Group of the resource
metav1.GroupVersionKind `json:",inline"`

// Type is the event type. Refer https://github.com/kubernetes/apimachinery/blob/dcb391cde5ca0298013d43336817d20b74650702/pkg/watch/watch.go#L43
// If not provided, the gateway will watch all events for a resource.
Type watch.EventType `json:"type,omitempty"`
```

### Example
The following gateway configmap contains a couple of event sources. 

The `successWorkflow` event source defines configuration to listen to notifications for newly created `Argo Workflows` and filter these
workflows on labels. 

The `namespace` event source defines configuration to listen to any updates to namespace.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: resource-gateway-configmap
data:
  workflow-success: |-
    namespace: argo-events
    group: "argoproj.io"
    version: "v1alpha1"
    kind: "Workflow"
    # https://github.com/kubernetes/apimachinery/blob/dcb391cde5ca0298013d43336817d20b74650702/pkg/watch/watch.go#L43
    type: ADDED
    filter:
      labels:
        workflows.argoproj.io/phase: Succeeded
        name: "my-workflow"
  namespace: |-
    namespace: argo-events
    group: "k8s.io"
    version: "v1"
    kind: "Namespace"
```

### Event Payload Structure
Kubernetes Object the gateway is watching.

## Setup

**1. Install Gateway Configmap**

```yaml
kubectl -n argo-events create -f  https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/resource-gateway-configmap.yaml
```

**2. Install gateway**

```yaml
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/gateways/resource.yaml
```

Make sure the gateway pod is created
   
**3. Install Sensor**

```yaml
kubectl -n argo-events create -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/sensors/resource.yaml
```

Make sure the sensor pod is created.

**4. Trigger workflow**

Create an argo workflow with name `my-workflow`. As soon as `my-workflow` succeeds, a new workflow will be triggered.

