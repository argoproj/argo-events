# Service Account for Triggers

Based on the type of trigger, it's a good practice to create a service account with minimum set of roles to execute it.

The sensor examples use `argo-events-sa` service account to execute all types of triggers, but it is has more permissions than needed
,and you may want to limit those permissions based on your use-case.

## K8s Resource Trigger

* To execute Argo workflow trigger, make sure to grant `create` permission for workflows to the service account.

* To trigger a any other K8s resource, make sure to grant `create` permission for that resource. 

## AWS Lambda, HTTP, Slack and OpenWhisk Trigger

These triggers may need access to secrets for access tokens/auth related configuration. Make
sure to grant `get` and `list` permissions for the secret resource.  

## NATS and Kafka Triggers

For NATS and Kafka, you **don't** need any K8s role associated with the service account.
