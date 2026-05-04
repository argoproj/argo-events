## Jetstream

[Jetstream](https://docs.nats.io/nats-concepts/jetstream) is the latest streaming server implemented by the NATS community, with improvements from the original NATS Streaming (which will eventually be deprecated).

A simplest Jetstream EventBus example:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstream:
    version:
      latest # Do NOT use "latest" but a specific version in your real deployment
      # See: https://argoproj.github.io/argo-events/eventbus/jetstream/#version
```

The example above brings up a Jetstream
[StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
with 3 replicas in the namespace.

## Properties

Check
[here](../APIs.md#argoproj.io/v1alpha1.JetStreamBus)
for the full spec of `jetstream`.

### version

The version number specified in the example above is the release number for the NATS server. We will support some subset of these as we've tried them out and only plan to upgrade them as needed. The list of available versions is managed by the controller manager ConfigMap, which can be updated to support new versions.

```
kubectl get configmap argo-events-controller-config -o yaml
```

Check [here](https://docs.nats.io/nats-concepts/jetstream/streams#configuration) for a list of configurable features per version.

### A more involved example

Another example with more configuration:

```
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstream:
    version: latest # Do NOT use "latest" but a specific version in your real deployment
    replicas: 5
    persistence: # optional
        storageClassName: standard
        accessMode: ReadWriteOnce
        volumeSize: 10Gi
    streamConfig: |             # see default values in argo-events-controller-config
      maxAge: 24h
    settings: |
      max_file_store: 1GB       # see default values in argo-events-controller-config
    startArgs:
      - "-D"                    # debug-level logs
```

## Security

For Jetstream, TLS is turned on for all client-server communication as well as between Jetstream nodes. In addition, for client-server communication we by default use password authentication (and because TLS is turned on, the password is encrypted).

## How it works under the hood

Jetstream has the concept of a Stream, and Subjects (i.e. topics) which are used on a Stream. From the documentation: “Each Stream defines how messages are stored and what the limits (duration, size, interest) of the retention are.” For Argo Events, we have one Stream called "default" with a single set of settings, but we have multiple subjects, each of which is named `default.<eventsourcename>.<eventname>`. Sensors subscribe to the subjects they need using durable consumers.

### Exotic

To use an existing JetStream service instead of having Argo Events manage one,
use `jetstreamExotic`. This is useful when you already have a NATS JetStream
cluster and want Argo Events to connect to it as a client.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstreamExotic:
    url: nats://my-nats-server:4222
    accessSecret:
      name: my-secret-name
      key: secret-key
    streamConfig: ""
```

#### Authentication

The `accessSecret` field references a Kubernetes Secret containing the
credentials used to authenticate with the NATS server. When specified, Argo
Events uses basic (password) authentication. The Secret key should contain the
**NATS password or token** as a plain string.

For example, create the Secret:

```bash
kubectl create secret generic nats-auth \
  --from-literal=password=my-nats-password \
  -n argo-events
```

Then reference it in the EventBus:

```yaml
spec:
  jetstreamExotic:
    url: nats://my-nats-server:4222
    accessSecret:
      name: nats-auth
      key: password
```

If your NATS server does not require authentication (e.g., running in a
service mesh that provides mTLS, or in a development environment), you can
omit the `accessSecret` field entirely.

#### TLS

When `tls` is configured, Argo Events connects to the NATS server over TLS:

```yaml
spec:
  jetstreamExotic:
    url: nats://my-nats-server:4222
    tls:
      caCertSecret:
        name: nats-tls
        key: ca.crt
      clientCertSecret:
        name: nats-tls
        key: tls.crt
      clientKeySecret:
        name: nats-tls
        key: tls.key
```

#### Stream Configuration

The `streamConfig` field allows you to override the default JetStream stream
settings (e.g., retention policy, max age). If left empty, Argo Events uses
its default stream configuration. See the
[NATS JetStream stream configuration](https://docs.nats.io/nats-concepts/jetstream/streams#configuration)
for available options.
