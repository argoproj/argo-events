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
[here](https://github.com/argoproj/argo-events/blob/master/api/event-bus.md#argoproj.io/v1alpha1.JetstreamBus)
for the full spec of `jetstream`.

### version

The version number specified in the example above is the release number for the NATS server. We will support some subset of these as we've tried them out and only plan to upgrade them as needed. To take a look at what that includes:

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

Jetstream has the concept of a Stream, and Subjects (i.e. topics) which are used on a Stream. From the documentation: "Each Stream defines how messages are stored and what the limits (duration, size, interest) of the retention are." For Argo Events, we have one Stream called "default" with a single set of settings, but we have multiple subjects, each of which is named `default.<eventsourcename>.<eventname>`. Sensors subscribe to the subjects they need using durable consumers.

### Sensor Deliver Policy Configuration

When using JetStream as the EventBus, you can configure the deliver policy for each sensor dependency using the `jetStream` field in the `EventDependency` specification. This allows you to control how messages are delivered to your sensor:

- **`all`**: Start receiving from the earliest available message in the stream
- **`last`**: Start with the last message added to the stream, or the last message matching the consumer's filter subject if defined
- **`new`**: Start receiving messages created after the consumer was created

Example sensor configuration:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: example
spec:
  dependencies:
    - name: test-dep
      eventSourceName: webhook
      eventName: example
      jetStream:
        deliverPolicy: "all" # Start from earliest available message (default)
  triggers:
    - template:
        name: http-trigger
        http:
          url: http://example.com/webhook
```

### Exotic

To use an existing JetStream service, follow the example below.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  jetstreamExotic:
    url: nats://xxxxx:xxx
    accessSecret:
      name: my-secret-name
      key: secret-key
    streamConfig: ""
```
