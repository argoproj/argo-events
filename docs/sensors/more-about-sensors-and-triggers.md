# More About Sensors And Triggers

## Multiple Dependencies

If there are multiple dependencies defined in the `Sensor`, you can configure
[Trigger Conditions](trigger-conditions.md) to determine what kind of situation
could get the trigger executed.

For example, there are 2 dependencies `A` and `B` are defined, then condition
`A || B` means an event from either `A` or `B` will execute the trigger.

What happens if `A && B` is defined? Assume before `B` has an event `b1`
delivered, `A` has already got events `a1` - `a10`, in this case, `a10` and `b1`
will be used to execute the trigger, and `a1` - `a9` will be dropped.

In short, at the moment `Trigger Conditions` resolve to true, the latest events
from each dependencies will be used to trigger the actions.

## Duplicate Dependencies

Due to technical reasons when using the NATS Streaming bus, the same `eventSourceName` and `eventName` combo can not
be referenced twice in one `Sensor` object. For example, the following dependency
definitions are not allowed. However, it can be referenced unlimited times in
different `Sensor` objects, so if you do have similar requirements, use 2
`Sensor` objects instead.

```yaml
spec:
  dependencies:
    - name: dep01
      eventSourceName: webhook
      eventName: example
      filters:
        data:
          - path: body.value
            type: number
            comparator: "<"
            value:
              - "20.0"
    - name: dep02
      eventSourceName: webhook
      eventName: example
      filters:
        data:
          - path: body.value
            type: number
            comparator: ">"
            value:
              - "50.0"
```

Note that this is not an issue for the Jetstream bus, however.

## Events Delivery Order

Following statements are based on using `NATS Streaming` as the EventBus.

In general, the order of events delivered to a `Sensor` is the order they were
published, but there's no guarantee for that. There could be cases that the
`Sensor` fails to acknowledge the first message, and then succeeds to
acknowledge the second one before the first one is redelivered.

## Events Delivery Guarantee

`NATS Streaming` offers `at-least-once` delivery guarantee. `Jetstream` has additional features that get closer to "exactly once". In addition, in the `Sensor` application, an in-memory cache is implemented to cache the events IDs delivered
in the last 5 minutes: this is used to make sure there won't be any duplicate
events delivered. Based on this, we are able to achieve 1) "exactly once" in almost all cases, with the exception of pods dying while processing messages, and 2) "at least once" in all cases.

## Trigger Retries

By default, there's no retry for the trigger execution, this is based on the
fact that `Sensor` has no idea if failure retry would bring any unexpected
results.

If you prefer to have retry for the `trigger`, add `retryStrategy` to the spec.

```yaml
spec:
  triggers:
    - template:
        name: http-trigger
        http:
          url: https://xxxxx.com/
          method: GET
      retryStrategy:
        # Give up after this many times
        steps: 3
```

Or if you want more control on the retries:

```yaml
spec:
  triggers:
    - retryStrategy:
        # Give up after this many times
        steps: 3
        # The initial duration, use strings like "2s", "1m"
        duration: 2s
        # Duration is multiplied by factor each retry, if factor is not zero
        # and steps limit has not been reached.
        # Should not be negative
        #
        # Defaults to "1.0"
        factor: 2.0
        # The sleep between each retry is the duration plus an additional
        # amount chosen uniformly at random from the interval between
        # zero and `jitter * duration`.
        #
        # Defaults to "1"
        jitter: 2
```

## Trigger Rate Limit

There's no rate limit for a trigger unless you configure the spec as following:

```yaml
spec:
  triggers:
    - rateLimit:
        # Second, Minute or Hour, defaults to Second
        unit: Second
        # Requests per unit
        requestsPerUnit: 20
```

## Revision History Limit

Optionally, a `revisionHistoryLimit` may be configured in the spec as following:

```yaml
spec:
  # Optional
  revisionHistoryLimit: 3
```
