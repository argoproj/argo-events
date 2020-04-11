# Parameterization

In previous section, we saw how to set up a basic webhook gateway and sensor, and trigger
an Argo workflow. The trigger template had `parameters` set in the sensor obejct and
the workflow was able to print the event payload. In this tutorial, we will dig deeper into
different types of parameterization, how to extract particular key-value from event payload
and how to use default values if certain `key` is not available within event payload.

## Trigger Resource Parameterization
If you take a closer look at the `Sensor` object, you will notice it contains a list
of triggers. Each `Trigger` contains the template that defines the context of the trigger
and actual `resource` that we expect the sensor to execute. In previous section, the `resource` within
the trigger template was an Argo workflow.

This subsection deals with how to parameterize the `resource` within trigger template
with the event payload.

### Prerequisites
Make sure to have the basic webhook gateway and sensor set up. Follow the [introduction](https://argoproj.github.io/argo-events/tutorials/01_introduction) tutorial if haven't done already.

### Webhook Event Payload
Webhook gateway consumes events through HTTP requests and transforms them into CloudEvents.
The structure of the event the Webhook sensor receives from the gateway looks like following,

        {
            "context": {
              "type": "type_of_gateway",
              "specVersion": "cloud_events_version",
              "source": "name_of_the_gateway",
              "eventID": "unique_event_id",
              "time": "event_time",
              "dataContentType": "type_of_data",
              "subject": "name_of_the_event_within_event_source"
            },
            "data": {
              "header": {},
              "body": {},
            }
        }

1. `Context`: This is the CloudEvent context and it is populated by the gateway regardless of 
type of HTTP request.

2. `Data`: Data contains following fields,
   1. `Header`: The `header` within event `data` contains the headers in the HTTP request that was dispatched
   to the gateway. The gateway extracts the headers from the request and put it in
   the the `header` within event `data`. 

   2. `Body`: This is the request payload from the HTTP request.

### Event Context
Now that we have an understanding of the structure of the event the webhook sensor receives from
the gateway, lets see how we can use the event context to parameterize the Argo workflow.

1. Update the `Webhook Sensor` and add the `contextKey` for the parameter at index 0.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/02-parameterization/sensor-01.yaml

2. Send a HTTP request to the gateway.

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. Inspect the output of the Argo workflow that was created.

        argo logs name_of_the_workflow
   
   You will see the following output,

        _________
        < webhook >
        ---------
           \
            \
             \
                           ##        .
                     ## ## ##       ==
                  ## ## ## ##      ===
              /""""""""""""""""___/ ===
         ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~
              \______ o          __/
               \    \        __/
                 \____\______/

We have successfully extracted the `type` key within the event context and parameterized
the workflow to print the value of the `type`.

### Event Data
Now, it is time to use the event data and parameterize the Argo workflow trigger.
We will extract the `message` from request payload and get the Argo workflow to 
print the message.

1. Update the `Webhook Sensor` and add the `dataKey` in the parameter at index 0.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/02-parameterization/sensor-02.yaml

2. Send a HTTP request to the gateway.

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. Inspect the output of the Argo workflow that was created.

        argo logs name_of_the_workflow

   You will see the following output,

         __________________________ 
        < this is my first webhook >
         -------------------------- 
            \
             \
              \     
                            ##        .            
                      ## ## ##       ==            
                   ## ## ## ##      ===            
               /""""""""""""""""___/ ===        
          ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
               \______ o          __/            
                \    \        __/             
                  \____\______/   


Yay!! The Argo workflow printed the message. You can add however many number of parameters
to update the trigger resource on the fly.

**_Note_**: If you define both the `contextKey` and `dataKey` within a parameter, then
the `dataKey` takes the precedence.

### Default Values
Each parameter comes with an option to configure the default value. This is specially
important when the `key` you defined in the parameter doesn't exist in the event.

1. Update the `Webhook Sensor` and add the `value` for the parameter at index 0.
   We will also update the `dataKey` to an unknown event key.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/02-parameterization/sensor-03.yaml

2. Send a HTTP request to the gateway.

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. Inspect the output of the Argo workflow that was created.

        argo logs name_of_the_workflow

   You will see the following output,

        _______________________ 
        < wow! a default value. >
        ----------------------- 
           \
            \
             \     
                           ##        .            
                     ## ## ##       ==            
                  ## ## ## ##      ===            
              /""""""""""""""""___/ ===        
         ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
              \______ o          __/            
               \    \        __/             
                 \____\______/   


<br/>

### Sprig Templates

The [sprig template](https://github.com/Masterminds/sprig) exposed through `contextTemplate` and `dataTemplate` lets you alter the event
context and event data before it gets applied to the trigger via `parameters`.

Take a look at the example defined [here](https://github.com/argoproj/argo-events/blob/master/examples/sensors/trigger-with-template.yaml), it contains the parameters
as follows,

        parameters:
        # Retrieve the 'message' key from the payload
        - src:
            dependencyName: test-dep
            dataTemplate: "{{ .Input.body.message | title }}"
          dest: spec.arguments.parameters.0.value
        # Title case the context subject
        - src:
            dependencyName: test-dep
            contextTemplate: "{{ .Input.subject | title }}"
          dest: spec.arguments.parameters.1.value
        # Retrieve the 'name' key from the payload, remove all whitespace and lowercase it.
        - src:
            dependencyName: test-dep
            dataTemplate: "{{ .Input.body.name | nospace | lower }}-"
          dest: metadata.generateName
          operation: append


Consider the event the sensor received has format like,

        {
            "context": {
              "type": "type_of_gateway",
              "specVersion": "cloud_events_version",
              "source": "name_of_the_gateway",
              "eventID": "unique_event_id",
              "time": "event_time",
              "dataContentType": "type_of_data",
              "subject": "name_of_the_event_within_event_source"
            },
            "data": {
              "body": {
                "name": "foo bar",
                "message": "hello there!!"
              },
            }
        }

The parameters are transformed as,

1. The first parameter extracts the `body.message` from event data and applies `title` filter which basically
   capitalizes the first letter and replaces the `spec.arguments.parameters.0.value`.
1. The second parameter extracts the `subject` from the event context and again applies `title` filter and replaces the
   `spec.arguments.parameters.1.value`.
1. The third parameter extracts the `body.name` from the event data, applies `nospace` filter which removes all
   white spaces and then `lower` filter which lowercases the text and appends it to `metadata.generateName`.

Send a curl request to webhook-gateway as follows,

        curl -d '{"name":"foo bar", "message": "hello there!!"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

and you will an Argo workflow being sprung with name like `webhook-foobar-xxxxx`.


Check the output of workflow, it should print something like,

         ____________________________ 
        < Hello There!! from Example >
         ---------------------------- 
            \
             \
              \     
                            ##        .            
                      ## ## ##       ==            
                   ## ## ## ##      ===            
               /""""""""""""""""___/ ===        
          ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
               \______ o          __/            
                \    \        __/             
                  \____\______/   

<br/>

### Operations
Sometimes you need the ability to append or prepend a parameter value to 
an existing value in trigger resource. This is where the `operation` field within
a parameter comes handy.

1. Update the `Webhook Sensor` and add the `operation` in the parameter at index 0.
   We will prepend the message to an existing value.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/02-parameterization/sensor-04.yaml


2. Send a HTTP request to the gateway.

        curl -d '{"message":"hey!!"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. Inspect the output of the Argo workflow that was created.

        argo logs name_of_the_workflow

   You will see the following output,

         __________________ 
        < hey!!hello world >
         ------------------ 
            \
             \
              \     
                            ##        .            
                      ## ## ##       ==            
                   ## ## ## ##      ===            
               /""""""""""""""""___/ ===        
          ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
               \______ o          __/            
                \    \        __/             
                  \____\______/   


## Trigger Template Parameterization
The parameterization you saw above deals with the trigger resource, but sometimes
you need to parameterize the trigger template itself. This comes handy when you have
the trigger resource stored on some external source like S3, Git, etc. and you need
to replace the url of the source on the fly in trigger template.

Imagine a scenario where you want to parameterize the parameters of trigger to 
parameterize the trigger resource. What?...

The sensor you have been using in this tutorial has one parameter defined in the
trigger resource under `k8s`. We will parameterize that `parameter` by
applying a parameter at the trigger template level.


1. Update the `Webhook Sensor` and add parameters at trigger level.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/02-parameterization/sensor-05.yaml

2. Send a HTTP request to the gateway.

        curl -d '{"dependencyName":"test-dep", "dataKey": "body.message", "dest": "spec.arguments.parameters.0.value", "message": "amazing!!"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

3. Inspect the output of the Argo workflow that was created.

        argo logs name_of_the_workflow

   You will see the following output,

        ___________ 
        < amazing!! >
        ----------- 
        \
         \
          \     
                        ##        .            
                  ## ## ##       ==            
               ## ## ## ##      ===            
           /""""""""""""""""___/ ===        
        ~~~ {~~ ~~~~ ~~~ ~~~~ ~~ ~ /  ===- ~~~   
           \______ o          __/            
            \    \        __/             
              \____\______/      


Great!! You have now learned how to apply parameters at trigger resource and template level.
Keep in mind that you can apply default values and operations like prepend and append for 
trigger template parameters as well.
