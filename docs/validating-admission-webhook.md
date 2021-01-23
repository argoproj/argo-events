# Validating Admission Webhook

![alpha](assets/alpha.svg)

> v1.3 and after

## Overview

Starting from v1.3, a
[Validating Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#validatingadmissionwebhook)
is introduced to the project. To install Argo Events with the validating
webhook, use following command (update the version):

```shell
kubectl apply -n argo-events -f https://raw.githubusercontent.com/argoproj/argo-events/{version}/manifests/install-with-extension.yaml
```

## Benefits

Using the validation webhook brings following benefits:

1. It notifies the error before the faulty spec is persisted into K8s etcd.

e.g. Creating a `exotic` NATS EventBus without `ClusterID` specified:

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
Error from server (BadRequest): error when creating "STDIN": admission webhook "webhook.argo-events.argoproj.io" denied the request: ClusterID is missing
```

2. It validates spec update behavior.

For example, updating Auth Strategy for a native NATS EventBus will get a denied
response as following:

```sh
Error from server (BadRequest): error when applying patch:
{"metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"argoproj.io/v1alpha1\",\"kind\":\"EventBus\",\"metadata\":{\"annotations\":{},\"name\":\"default\",\"namespace\":\"argo-events\"},\"spec\":{\"nats\":{\"native\":{\"replicas\":3}}}}\n"}},"spec":{"nats":{"native":{"auth":null,"maxAge":null,"securityContext":null}}}}
to:
Resource: "argoproj.io/v1alpha1, Resource=eventbus", GroupVersionKind: "argoproj.io/v1alpha1, Kind=EventBus"
Name: "default", Namespace: "argo-events"
for: "test-eventbus.yaml": admission webhook "webhook.argo-events.argoproj.io" denied the request: Auth strategy is not allowed to be updated
```
