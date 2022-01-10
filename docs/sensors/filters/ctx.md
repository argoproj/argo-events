# Context Filter

Similar to the data filter, you can apply a filter on the context of the event.

Change the subscriber in the webhook event-source to point it to `context-filter` sensor's URL.

1. Lets create a webhook sensor with context filter.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/sensors/filter-with-context.yaml
   ```

1. Send a HTTP request to event-source.

   ```bash
   curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```

1. You will notice that the sensor logs prints the event is invalid as the sensor expects for
   either `custom-webhook` as the value of the `source`.

## Further examples

You can find some examples [here](https://github.com/argoproj/argo-events/tree/master/examples/sensors).
