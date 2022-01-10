# Data Filter

Data filters as the name suggests are applied on the event data. A CloudEvent from Webhook event-source has
payload structure as:

```json
{
    "context": {
      "type": "type_of_event_source",
      "specversion": "cloud_events_version",
      "source": "name_of_the_event_source",
      "id": "unique_event_id",
      "time": "event_time",
      "datacontenttype": "type_of_data",
      "subject": "name_of_the_configuration_within_event_source"
    },
    "data": {
      "header": {},
      "body": {},
    }
}
```

Data Filters are applied on `data` within the payload. We will make a simple HTTP request
to webhook event-source with request data as `{"message":"this is my first webhook"}` and apply
data filter on `message`.

A data filter has following fields:

```yaml
filters:
  data:
    - path: path_within_event_data
      type: types_of_the_data
      value:
        - list_of_possible_values
```

> ⚠️ `PLEASE NOTE` order in which filters are declared corresponds to the order in which the Sensor will evaluate them.

### Comparator

The data filter offers `comparator` “>=”, “>”, “=”, “!=”, “<”, or “<=”.

e.g.

```yaml
filters:
  data:
    - path: body.value
      type: number
      comparator: ">"
      value:
        - "50.0"
```

<br/>

**Note**: If data type is a `string`, then you can pass either an exact value or a regex.
If data types is bool or float, then you need to pass the exact value.

1. Lets create a webhook sensor with data filter.

  ```bash
  kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/07-filters/sensor-data-filters.yaml
  ```

1. Send a HTTP request to event-source.

  ```bash
  curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
  ```

1. You will notice that the sensor logs prints the event is invalid as the sensor expects for
   either `hello` or `hey` as the value of `body.message`.

1.  Send a valid HTTP request to event-source.

  ```bash
  curl -d '{"message":"hello"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
  ```

1. Watch for a workflow with name `data-workflow-xxxx`.

### Multiple Paths

If the HTTP request was less simple and contained multiple paths that we would like to filter against,
we can make use of [multipaths](https://github.com/tidwall/gjson/blob/master/SYNTAX.md#multipaths) to combine
multiple data paths in the payload into one string.

For a given payload such as:

```json
{
  "body": {
    "action":"opened",
   "labels": [
      {"id":"1234", "name":"Webhook"}, 
      {"id":"5678", "name":"Approved"}
    ]
  }
}
```

We want our sensor to fire if the action is "opened" and it has a label of "Webhook" or if the action is "closed"
and it has a label of "Webhook" and "Approved". We could therefore define the path as:

```yaml
filters:
  data:
    - path: "[body.action,body.labels.#(name=="Webhook").name,body.labels.#(name=="Approved").name]"
      type: string
# ...
```

This would return a string like: `["opened","Webhook","Approved"]`. As the resulting data type will be a
`string`, we can pass a regex over it:

```yaml
filters:
  data:
    - path: "[body.action,body.labels.#(name=="Webhook").name,body.labels.#(name=="Approved").name]"
      type: string
      value:
        - "(\bopened\b.*\bWebhook\b)|(\blabeled\b.*(\bWebhook\b.*\bApproved\b))"
```

### Template

The data filter offers `template`.
`template` process the incoming data defined in `path` through [sprig template](https://github.com/Masterminds/sprig) 
before matching with the `value`.

e.g.

```yaml
filters:
  data:
    - path: body.message
      type: string
      value:
        - "hello world"
      template: "{{ b64dec .Input }}"
```

message `'{"message":"aGVsbG8gd29ybGQ="}'` will match with the above filter definition.

**Note**: Data type is assumed to be string before applying the `template`, then cast to the user defined `type` for 
value matching.

## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
