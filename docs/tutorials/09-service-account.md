## Service Account for EventSources

Most of the event-sources can be run with a service account with no roles associated, expect `Resource` event-source. 
You need to associate the `get`, `list` and `watch` permissions for the resource being watched, and assign that role to the service account. 

## Service Account for Triggers

Based on the type of trigger, it's a good practice to create a service account with minimum set of roles to execute it.

The sensor examples use `argo-events-sa` service account to execute all types of triggers, but it is has more permissions than needed
,and you may want to limit those permissions based on your use-case.

### K8s Resource Trigger

* To execute Argo workflow trigger, make sure to grant `create` permission for workflows to the service account.

* To trigger a any other K8s resource, make sure to grant `create` permission for that resource. 

### AWS Lambda, HTTP, Slack and OpenWhisk Trigger, NATS and Kafka Triggers

For these triggers, you **don't** need any K8s role associated with the service account.

