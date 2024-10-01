# Policy

A policy for a trigger determines whether the trigger resulted in success or failure.

Currently, Argo Events supports 2 types of policies:

1. Policy based on the K8s resource labels.
2. Policy based on the response status for triggers like HTTP request, AWS Lambda, etc.

## Resource Labels Policy

This type of policy determines whether trigger completed successfully based on the labels
set on the trigger resource.

Consider a sensor which has an Argo workflow as the trigger. When
an Argo workflow completes successfully, the workflow controller sets a label on the resource as `workflows.argoproj.io/completed: 'true'`.
So, in order for sensor to determine whether the trigger workflow completed successfully,
you just need to set the policy labels as `workflows.argoproj.io/completed: 'true'` under trigger template.

In addition to labels, you can also define a `backoff` and option to error out if sensor
is unable to determine status of the trigger after the backoff completes. Check out the specification of
resource labels policy [here](../APIs.md#argoproj.io/v1alpha1.K8SResourcePolicy).

## Status Policy

For triggers like HTTP request or AWS Lambda, you can apply the `Status Policy` to determine the trigger status.
The Status Policy supports list of expected response statuses. If the status of the HTTP request or Lambda is within
the statuses defined in the policy, then the trigger is considered successful.

Complete specification is available [here](../APIs.md#argoproj.io/v1alpha1.StatusPolicy).
