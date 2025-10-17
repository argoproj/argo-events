# Validating Admission Webhook

![alpha](assets/alpha.svg)

> v1.3 and after

## Overview

Starting from v1.3, a
[Validating Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook)
is introduced to the project. To install the validating webhook, use following
command (change the version):

```shell
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/{version}/manifests/install-validating-webhook.yaml
```

## Benefits

Using the validating webhook has the following benefits:

- It notifies the error at the time applying the faulty spec, so that you don't
  need to check the CRD object `status` field to see if there's any condition
  errors later on.

  For example, creating an `exotic` NATS EventBus without `ClusterID` specified:

```sh
cat <<EOF | kubectl create -f -
> apiVersion: argoproj.io/v1alpha1
> kind: EventBus
> metadata:
>   name: default
> spec:
>   nats:
>     exotic: {}
> EOF
Error from server (BadRequest): error when creating "STDIN": admission webhook "webhook.argo-events.argoproj.io" denied the request: "spec.nats.exotic.clusterID" is missing
```

- Spec updating behavior can be validated.

  Updating existing specs requires more validation. Besides checking if the new
  spec is valid, we also need to check if any immutable fields are being
  updated. This cannot be done in the controller reconciliation, but we can do
  it by using the validating webhook.

  For example, updating the Auth Strategy for a native NATS EventBus is prohibited.
  A denied response as shown below will be returned.

```sh
Error from server (BadRequest): error when applying patch:
{"metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"argoproj.io/v1alpha1\",\"kind\":\"EventBus\",\"metadata\":{\"annotations\":{},\"name\":\"default\",\"namespace\":\"argo-events\"},\"spec\":{\"nats\":{\"native\":{\"replicas\":3}}}}\n"}},"spec":{"nats":{"native":{"auth":null,"maxAge":null,"securityContext":null}}}}
to:
Resource: "argoproj.io/v1alpha1, Resource=eventbus", GroupVersionKind: "argoproj.io/v1alpha1, Kind=EventBus"
Name: "default", Namespace: "argo-events"
for: "test-eventbus.yaml": admission webhook "webhook.argo-events.argoproj.io" denied the request: "spec.nats.native.auth" is immutable, can not be updated
```
