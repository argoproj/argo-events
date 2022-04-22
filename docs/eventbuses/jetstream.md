## Jetstream

A simplest Jetstream EventBus example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstream:
    version: 2.8.1  # see 'version' section below
```

The example above brings up a Jetstream
[StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
with 3 replicas in the namespace. 

### version

Note that the version number specified is the release number for the NATS server. We will support some subset of these as we've tried them out and only plan to upgrade them as needed. To take a look at what that includes:
```
kubectl describe configmap argo-events-controller-config
```

### Security

For Jetstream, TLS is turned on for all client-server communication as well as between Jetstream nodes. In addition, for client-server communication we by default use password authentication (and because TLS is turned on, the password is encrypted).

### Persistence

Example with persistence turned on:
```
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstream:
    version: 2.8.1
    persistence: # optional
        storageClassName: standard
        accessMode: ReadWriteOnce
        volumeSize: 10Gi
```

### Properties

Check
[here](https://broken)
for the full spec of `jetstream`.


