# Examples

The examples demonstrate how Argo Events works.

To make the Sensors be able to trigger Workflows, a Service Account with RBAC
settings as follows is required (assume you run the examples in the namespace
`argo-events`).

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: argo-events
  name: operate-workflow-sa
---
# Similarly you can use a ClusterRole and ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: operate-workflow-role
  namespace: argo-events
rules:
  - apiGroups:
      - argoproj.io
    verbs:
      - "*"
    resources:
      - workflows
      - workflowtemplates
      - cronworkflows
      - clusterworkflowtemplates
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: operate-workflow-role-binding
  namespace: argo-events
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: operate-workflow-role
subjects:
  - kind: ServiceAccount
    name: operate-workflow-sa
```

To make the Workflow triggered by the Sensor work, you also need to give a
Service Account with privileges to the Workflow (the examples use Service
Account `default`), see the details
[here](https://github.com/argoproj/argo-workflows/blob/master/docs/service-accounts.md).
A minimal Role to make Workflow work looks like following (check the
[origin](https://github.com/argoproj/argo-workflows/blob/master/docs/workflow-rbac.md)):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: workflow-role
rules:
  # pod get/watch is used to identify the container IDs of the current pod
  # pod patch is used to annotate the step's outputs back to controller (e.g. artifact location)
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
      - watch
      - patch
  # logs get/watch are used to get the pods logs for script outputs, and for log archival
  - apiGroups:
      - ""
    resources:
      - pods/log
    verbs:
      - get
      - watch
```

The Workflow triggered by the Sensor defaults to be in the same namespace as the
Sensor, if you want to trigger it in a different namespace, simply give a
`namespace` in the workflow metadata (in that case, a `ClusterRole` and
`ClusterRoleBinding` are required for `operate-workflow-sa`).
