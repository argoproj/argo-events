# Filtering EventSources

When event sources watch events from external data sources (ie. Kafka topics), it will ingest all messages.
With filtering, we are able to apply constraints and determine if the event should be published or skipped.
This is achieved by evaluating an expression in the EventSource spec.

# Fields

A `filter` in an example Kafka EventSource:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: kafka
spec:
  kafka:
    example:
      url: kafka.argo-events:9092
      topic: topic-2
      jsonBody: true
      partition: "1"
      filter: # filter field 
        expression: "(body.id == 4) && (body.name != 'Joe')" #expression to be evaluated
      connectionBackoff:
        duration: 10s
        steps: 5
        factor: 2
        jitter: 0.2
```

The `expression` string is evaluated with the [expr](https://github.com/antonmedv/expr) package which offers a wide set of basic operators and comparators.

# Example

1. Creating a Kafka EventSource with filter field present

```
kubectl apply -f examples/event-sources/kafka.yaml -n argo-events
```

2. Sending an event with passing filter conditions to kafka

```
echo '{"id": 4,"name": "John", "email": "john@intuit.com", "department":{"id": 1,"name": "HR","bu":{"id": 2,"name" : "devp"}}}' | kcat -b localhost:9092 -P -t topic-2
```

3. Sending an event with failing filter conditions

```
echo '{"id": 2,"name": "Johnson", "email": "john@intuit.com", "department":{"id": 1,"name": "HR","bu":{"id": 2,"name" : "devp"}}}' | kcat -b localhost:9092 -P -t topic-2
```

# Output

Successful logs from kafka event source pod:

```
{"level":"info","ts":1644017495.0711913,"logger":"argo-events.eventsource","caller":"kafka/start.go:217","msg":"dispatching event on the data channel...","eventSourceName":"kafka","eventSourceType":"kafka","eventName":"example","partition-id":"0"}
{"level":"info","ts":1644017495.1374986,"logger":"argo-events.eventsource","caller":"eventsources/eventing.go:514","msg":"succeeded to publish an event","eventSourceName":"kafka","eventName":"example","eventSourceType":"kafka","eventID":"kafka:example:kafka-broker:9092:topic-2:0:7"}
```
