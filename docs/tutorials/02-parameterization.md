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

  ```json
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
  ``` 

1. `Context`: This is the CloudEvent context and it is populated by the gateway regardless of 
type of HTTP request.

2. `Data`: Data contains following fields,
   1. `Header`: The `header` within event `data` contains the headers in the HTTP request that was dispatched
   to the gateway. The gateway pretty much takes the headers from the request in receives and put it in
   the the `header` within event `data`. 

   2. `Body`: This is the request payload from the HTTP request.

### Event Context
Now we have an understanding of the structure of the event the webhook sensor receives from
the gateway, lets see how we can use the event context to parameterize the Argo workflow.

1. Update the `Webhook Sensor` and add the `contextKey` in the parameter at index 0.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/sensor-01.yaml
   ```  

2. Send a HTTP request to the gateway.

   ```bash
   curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```

3. Inspect the output of the Argo workflow that was created.

   ```bash
   argo logs name_of_the_workflow
   ```
   
   You will see following output,
  
   ```json
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
   ```


We have successfully extracted the `type` key within the event context and parameterized
the workflow to print the value of the `type`.

