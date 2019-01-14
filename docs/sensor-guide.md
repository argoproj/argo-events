## Sensor Guide

Sensors define a set of dependencies (inputs) and actions (outputs). The sn's actions will only be triggered after it's dependencies have been resolved.
[Sensor examples](https://github.com/argoproj/argo-events/tree/master/examples/sensors)


### Dependencies
Dependencies(also called signals) are defined as "gateway-name/specific-configuration"
``` 
signals:
    - name: calendar-gateway/calendar.fooConfig
    - name: webhook-gateway/webhook.barConfig
```

### Repeating the sn
Sensor can be configured to rerun by setting repeat property to `true`
``` 
spec:
  repeat: true
```

### Triggers
Refer [Triggers](trigger-guide.md) guide.


### Event payload
Event payload of any signal can be passed to any trigger. To pass complete payload without applying any filter,
do not set ```path```
e.g.
```yaml
parameters:
  - src:
      signal: webhook-gateway/webhook.fooConfig
      path:
    dest: spec.templates.0.container.args.0
``` 

Complete example to pass payload from signal to trigger can be found [here](https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook.yaml) 

To pass a particular field from payload to trigger, set ```path```. e.g.
```yaml
parameters:
  - src:
      signal: webhook-gateway/webhook.fooConfig
      path: name
    dest: spec.templates.0.container.args.0
```

In above example, the object/value corresponding to key ```name``` will be passed to trigger.  

### Filters
Additionally, you can apply filters on the payload.

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