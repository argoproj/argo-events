# Parameterize triggers

## How to pass an event payload to a trigger?
Payload of an event can be passed to any trigger. The way it works is you define `parameters` on trigger resource in sensor spec. 
The `parameters` define list of

   1. `src`:
       1. `event`: the name of the event dependency
       2. `path`: which is basically a key within event payload to look for
       3. `value`: which is a default value if sensor can't find `path` in event payload.   
   2. `dest`: destination key within trigger spec whose corresponding value needs to be replaced with value from event payload
  
* Pass complete event payload to trigger [example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-with-complete-payload.yaml)
* Extract particular value from event payload and pass to trigger [example](https://github.com/argoproj/argo-events/blob/master/examples/sensors/webhook-with-resource-param.yaml) 
