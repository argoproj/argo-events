# Service Accounts

## Service Account for EventSources

A `Service Account` can be specified in the EventSource object with
`spec.template.serviceAccountName`, however it is not needed for all the
EventSource types except `resource`. For a `resource` EventSource, you need to
specify a Service Accout and give it `list` and `watch` permissions for the
resource being watched.

For example, if you want to watch actions on `Deployment` objects, you need to:

1.  Create a Service Account.

        kubectl -n your-namespace create sa my-sa

2.  Grant RBAC privileges to it.

        kubectl -n your-namespace create role deployments-watcher --verb=list,watch --resource=deployments.apps

        kubectl -n your-namespace create rolebinding deployments-watcher-role-binding --role=deployments-watcher --serviceaccount=your-namespace:my-sa

    or (if you want to watch cluster scope)

        kubectl create clusterrole deployments-watcher --verb=list,watch --resource=deployments.apps

        kubectl create clusterrolebinding deployments-watcher-clusterrole-binding --clusterrole=deployments-watcher --serviceaccount=your-namespace:my-sa

## Service Account for Sensors

A `Service Account` also can be specified in a Sensor object via
`spec.template.serviceAccountName`, this is only needed when `k8s` trigger or
`argoWorkflow` trigger is defined in the Sensor object.

The sensor examples provided by us use `operate-workflow-sa` service account to
execute the triggers, but it has more permissions than needed, and you may want
to limit those privileges based on your use-case. It's always a good practice to
create a service account with minimum privileges to execute it.

### Argo Workflow Trigger

- To `submit` a workflow through `argoWorkflow` trigger, make sure to grant the
  Service Account `create` and `list` access to `workflows.argoproj.io`.

- To `resubmit`, `retry`, `resume` or `suspend` a workflow through
  `argoWorkflow` trigger, the service account needs `update` and `get` access to
  `workflows.argoproj.io`.

### K8s Resource Trigger

To trigger a K8s resource including `workflows.argoproj.io` through `k8s`
trigger, make sure to grant `create` permission to that resource.

### AWS Lambda, HTTP, Slack, NATS, Kafka, and OpenWhisk Triggers

For these triggers, you **don't** need to specify a Service Account to the
Sensor.

## Service Account for Trigged Workflows (or other K8s resources)

When the Sensor is used to trigger a Workflow, you might need to configure the
Service Account used in the Workflow spec (**NOT**
`spec.template.serviceAccountName`) following Argo Workflow
[instructions](https://github.com/argoproj/argo-workflows/blob/master/docs/service-accounts.md).

If it is used to trigger other K8s resources (i.e. a Deployment), make sure to
follow least privilege principle.
