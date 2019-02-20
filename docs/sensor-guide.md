# Sensor Guide
Sensors define a set of dependencies (inputs) and actions (outputs). The sensor's actions will only be triggered after it's dependencies have been resolved.

<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/sensor.png?raw=true" alt="Sensor"/>
</p>

<br/>

1. [Specifications](https://github.com/argoproj/argo-events/blob/master/docs/sensor-protocol.md)
2. [Terminology](#terminology)
3. [How it works?](#how-it-works)
4. [How to pass an event payload to a trigger?](#how-to-pass-an-event-payload-to-a-trigger)
5. [Examples](#examples)

## Terminology

  * ### What is a event dependency?
    A dependency is an event sensor is waiting to happen. It is defined as "gateway-name:event-source-name".

  * ### What is a dependency group?
    A dependency group is basically a group of dependencies.

  * ### What is a circuit?
    Circuit is any arbitrary boolean logic that can be applied on dependency groups.

  * ### What is a trigger?
    Refer [Triggers](trigger-guide.md).

## How it works?
  1. Once the sensor receives an event from gateway either over HTTP or through NATS, it validates
  the event against dependencies defined in sensor spec. If the event is expected, then it is marked as a valid
  event and the dependency is marked as resolved.
    
  2. If you haven't defined dependency groups, sensor basically waits for all dependencies to resolve
  and then kicks off triggers in sequence. If filters are defined, sensor applies the filter on target event 
  and if events pass the filters, triggers are fired.
  
  3. If you have defined dependency groups, sensor upon receiving an event evaluates the group/s to which the event belongs to
  and marks groups as resolved if all other event dependencies in these groups are already satisfied.
  
  4. Whenever a dependency group is resolved, sensor evaluates the `circuit` defined in spec. If the `circuit` resolves to true, the
  triggers are fired.
  
  5. Sensor always waits for `circuit` to resolve to true before firing triggers.
  
  5. You may not want to fire all triggers every time but rather selected triggers only when some of the dependency groups are resolved.
  For that, sensor offers `when` switch on triggers. Basically `when` switch is way to control when to fire certain trigger depending upon which dependency group is resolved. 

  6. After sensor fires triggers, it initialized it's state back to running and start the process all over again. Any event that is received in-between are stored on the internal queue.

  **Note**: If you don't provide dependency groups and `circuit`, sensor performs an `AND` operation on event dependencies.

## Filters
There are 3 types of filters:

|   Type   |   Description      |
|----------|-------------------|
|   **Time**            |   Filters the signal based on time constraints     |
|   **EventContext**    |   Filters metadata that provides circumstantial information about the signal.      |
|   **Data**            |   Describes constraints and filters for payload      |

<br/>

 * ### Time Filter
    ``` 
    filters:
            time:
              start: "2016-05-10T15:04:05Z07:00"
              stop: "2020-01-02T15:04:05Z07:00"
    ```

    [Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/time-filter-webhook.yaml)

 * ### EventContext Filter
    ``` 
    filters:
            context:
                source:
                    host: amazon.com
                contentType: application/json
    ```

    [Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/context-filter-webhook.yaml)

 * ### Data filter
    ```
    filters:
            data:
                - path: bucket
                  type: string
                  value: argo-workflow-input
    ```
    
    [Example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/data-filter-webhook.yaml)



## How to pass an event payload to a trigger?
Payload of an event can be passed to any trigger. The way it works is you define `parameters` on trigger resource in sensor spec. 
The `parameters` define list of  
  1. `src` -
     1. `event`: the name of the event dependency
     2. `path`: which is basically a key within event payload to look for
     3. `value`: which is a default value if sensor can't find `path` in event payload.   
  2. `dest`: destination key within trigger spec whose corresponding value needs to be replaced with value from event payload
  
* Pass complete event payload to trigger [example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-with-complete-payload.yaml)
* Extract particular value from event payload and pass to trigger [example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-with-resource-param.yaml) 

## Examples
You can find sensor examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors)

