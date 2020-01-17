# Special Argo Workflow Trigger

Although you can trigger an Argo workflow using a standard K8s trigger, the functionality provided by `argo` cli can't 
be leveraged in a standard K8s trigger. The special argo workflow trigger supports following operations for 
a workflow,

1. Submit
2. Resubmit
3. Resume
4. Retry
5. Suspend

The trigger specification is available [here](https://github.com/argoproj/argo-events/blob/worflow-triggers/api/sensor.md#argoworkflowtrigger) and the example is located under `examples/sensors`.
