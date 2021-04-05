# Sensor High Availability

Sensor controller creates a k8s deployment (replica number defaults to 1) for
each Sensor object. HA with `Active-Passive` strategy can be achieved by setting
`spec.replicas` to a number greater than 1, which means only one Pod serves
traffic and the rest ones stand by. One of standby Pods will be automatically
elected to be active if the old one is gone.

**Please DO NOT manually scale up the replicas, that might cause unexpected
behaviors!**

## RBAC

To achieve HA for Sensor Pods, a Service Account with extra RBAC settings is
needed. The Service Account needs to be bound to a Role like following, and
specified in the spec through `spec.template.serviceAccountName`.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: lease-role
rules:
  - apiGroups:
      - coordination.k8s.io
    resources:
      - leases
    resourceNames:
      - sensor-{sensor-name)
    verbs:
      - "*"
```

**NOTE: This is not requried if `spec.replicas = 1`.**

## More

Check [this](../dr_ha_recommendations.md) out to learn more information about
DR/HA.
