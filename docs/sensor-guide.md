# Sensor Guide
Sensors define a set of dependencies (inputs) and actions (outputs). The sensor's actions will only be triggered after it's dependencies have been resolved.
[Sensor examples](https://github.com/argoproj/argo-events/tree/master/examples/sensors)

1. [Specifications](#specifications)
2. [Dependencies](#dependencies)
3. [Triggers](#triggers)
4. [Passing event payload to trigger/s](#passing-event-payload-to-triggers)
5. [Filters](#filters)

## Specifications
https://github.com/argoproj/argo-events/blob/master/docs/sensor-protocol.md

## Dependencies
A sensor can contain list of event dependencies and a dependency is defined as "gateway-name:event-source-name"

## Triggers
Refer [Triggers](trigger-guide.md) guide.

## Passing event payload to trigger/s
Event payload of any signal can be passed to any trigger.

* Pass complete event payload https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-with-complete-payload.yaml
* Extract particular key from event payload and pass to trigger https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-with-resource-param.yaml 

## Filters
Additionally, you can apply filters on the payload.

There are 3 types of filters:

|   Type   |   Description      |
|----------|-------------------|
|   Time            |   Filters the signal based on time constraints     |
|   EventContext    |   Filters metadata that provides circumstantial information about the signal.      |
|   Data            |   Describes constraints and filters for payload      |

### Time Filter
``` 
filters:
        time:
          start: "2016-05-10T15:04:05Z07:00"
          stop: "2020-01-02T15:04:05Z07:00"
```

### EventContext Filter
``` 
filters:
        context:
            source:
                host: amazon.com
            contentType: application/json
```

### Data filter
```
filters:
        data:
            - path: bucket
              type: string
              value: argo-workflow-input
```