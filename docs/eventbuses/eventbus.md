# EventBus

![GA](../assets/ga.svg)

> v0.17.0 and after

EventBus is a Kubernetes
[Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
which is used for event transmission from EventSources to Sensors. Currently,
EventBus is backed by [NATS](https://docs.nats.io/), including both their NATS Streaming service and their newer Jetstream service. In the future, this can be expanded to support other technologies as well.

EventBus is namespaced; an EventBus object is required in a namespace to make
EventSource and Sensor work.

The common practice is to create an EventBus named `default` in the namespace. If
you want to use a different name, or you want to have multiple EventBuses in one
namespace, you need to specify `eventBusName` in the spec of EventSource and
Sensor correspondingly, so that they can find the right one. See EventSource
[spec](https://github.com/argoproj/argo-events/tree/stable/api/event-source.md#eventsourcespec)
and Sensor
[spec](https://github.com/argoproj/argo-events/tree/stable/api/sensor.md#sensorspec).



## More Information

- To view a finalized EventBus config:

```sh
kubectl get eventbus default  -o json | jq '.status.config'
```

A sample result:

```json
{
  "nats": {
    "accessSecret": {
      "key": "client-auth",
      "name": "eventbus-default-client"
    },
    "auth": "token",
    "clusterID": "eventbus-default",
    "url": "nats://eventbus-default-stan-svc:4222"
  }
}
```

- All the events in a namespace are published to same channel/subject/topic
  named `eventbus-{namespace}` in the EventBus.
