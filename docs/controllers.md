## Controllers

* Sensor and Gateway controllers are the components which manage Sensor and Gateway objects respectively. 

* Sensor and Gateway are Kubernetes Custom Resources. For more information on K8 CRDs visit [here.](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)

### Controller Configmap
Defines the `instance-id` and the `namespace` for the controller.
e.g. 
```yaml
# The gateway-controller configmap includes configuration information for the gateway-controller
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-controller-configmap
data:
  config: |
    instanceID: argo-events  # mandatory
    namespace: my-custom-namespace # optional
```

<b>`namespace`</b>: If you don't provide namespace, controller will watch all namespaces for gateway resource.

<b>`instanceID`</b>: it is used to map a gateway or sensor object to a controller. 
e.g. when you create a gateway with label `gateways.argoproj.io/gateway-controller-instanceid: argo-events`, a
 controller with label `argo-events` will process that gateway.

`instanceID` is used to horizontally scale controllers, so you won't end up overwhelming a single controller with large
 number of gateways or sensors. Also keep in mind that `instanceID` has nothing to do with namespace where you are
 deploying controllers and gateways/sensors objects.
