# Trigger Parameterization

Payload of an event can be passed to any trigger. Argo Events offers parameterization at two levels,

1) Template

2) Resource

## Template
You can parameterize the template that refers the Argo Worflow / K8s artifact. 

## Resource
Allows to parameterize the Argo workflow/K8s resource definition.


You can find an example [here.](https://github.com/argoproj/argo-events/blob/master/examples/sensors/complete-trigger-parameterization.yaml)

## Parameter Structure
A `parameter` contains:

   1. `src`:
       1. `event`: the name of the event dependency
       2. `path`: a key within event payload to look for
       3. `value`: default value if sensor can't find `path` in event payload
   2. `dest`: destination key within resource definition whose corresponding value needs to be replaced with value from event payload
   3. `operation`: what to do with the existing value at `dest`, either `overwrite`, `prepend`, or `append`
