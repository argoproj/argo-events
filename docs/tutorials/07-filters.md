# Filters

In previous sections, you have seen how to trigger an Argo workflow based on events. In this tutorial,
you will learn how to apply filters on event data and context. Filters provide you mechanism to
apply constraints on the events in order to determine a valid event. 

Argo Events offers 3 types of filters:

1. Data Filters
2. Context Filters
3. Time Filters

## Prerequisite
Webhook gateway must be set up.

## Data Filters
Data filters as the name suggests are applied on the event data. A CloudEvent from Webhook gateway has
payload structure as,

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

Data Filters are applied on `data` within the payload. We will make a simple HTTP request
to webhook gateway with request data as `{"message":"this is my first webhook"}` and apply
data filters on `message`.

A data filter has following fields,

```yaml
data:
  - path: path_within_event_data
    type: types_of_the_data
    value:
      - list_of_possible_values      
``` 

If data type is a `string`, then you can pass either an exact value or a regex.
If data types is either bool or float, then you need to pass the exact value.

1. Lets create a webhook sensor with data filters.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/07-filters/sensor-data-filters.yaml
   ```

2. Send a HTTP request to gateway

   ```bash
   curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```

3. You will notice that the sensor logs prints that the event was invalid as we are looking for
   either `hello` or `hey` as the value of `body.message`.
 
4.  Send a valid HTTP request to gateway
    
    ```bash
    curl -d '{"message":"hello"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
    ```

5. You will a workflow with name `data-workflow-xxxx` being created.

 