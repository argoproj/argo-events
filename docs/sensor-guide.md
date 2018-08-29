## Sensor Guide

Sensors define a set of dependencies (inputs) and actions (outputs). The sensor's actions will only be triggered after it's dependencies have been resolved.
[Checkout sensor examples](https://github.com/argoproj/argo-events/tree/eventing/examples/sensors)


### Dependencies
Dependencies(also called signals) are the name/s of gateway/s from whom the sensor expects to get the notifications.
``` 
signals:
    - name: calendar-gateway
    - name: foo-gateway
    - name: bar-gateway
```

### Triggers
Refer [Triggers](trigger-guide.md) guide.

### Repeating the sensor
Sensor can be configured to rerun by setting repeat property to `true`
``` 
spec:
  repeat: true
  signals:
    - name: calendar-example-gateway
```

### Filters
The payload received from gateways to trigger resources as input. You can apply filters
on the payload.

There are 3 types of filters:

|   Type   |   Description      |
|----------|-------------------|
|   Time            |   Filters the signal based on time constraints     |
|   EventContext    |   Filters metadata that provides circumstantial information about the signal.      |
|   Data            |   Describes constraints and filters for payload      |

#### Time Filter
``` 
filters:
        time:
          start: "2016-05-10T15:04:05Z07:00"
          stop: "2020-01-02T15:04:05Z07:00"
```

#### EventContext Filter
``` 
filters:
        context:
            source:
                host: amazon.com
            contentType: application/json
```

#### Data filter
```
filters:
        data:
            - path: bucket
              type: string
              value: argo-workflow-input
```