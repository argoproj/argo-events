# HTTP Trigger

Sometimes you face a situation where creating an Argo workflow on every event 
is not an ideal solution. This is where the HTTP trigger can help you. With this type of trigger, you can
connect any old/new API server with 20+ event sources supported by Argo Events or invoke serveless fucntions
without worrying about their respective event connector frameworks.

## Prerequisite
Set up the `Minio` gateway and event source. The K8s manifests are available under `examples/tutorials/09-http-trigger`.

## API servers Integration
We will set up a basic go http server and connect it with the minio events.

1. Set up a simple http server that prints the request payload.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/09-http-trigger/http-server.yaml
   ```

2. Create a service to expose the http server

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/09-http-trigger/http-server-svc.yaml
   ```

3. Either use Ingress, OpenShift Route or port-forwarding to expose the http server. We will
   use port-forwarding here.

4. Create a sensor with `HTTP` trigger. We will discuss the trigger details in the following sections.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/09-http-trigger/sensor-01.yaml
   ```

5. Now, drop a file onto `test` bucket in Minio server.

6. The sensor has triggered a http request to out basic go http server. Take a look at the logs

   ```bash
   server is listening on 8090
   {"type":"minio","bucket":"test"}
   ```

7. Great!!! But how did the sensor constructed a payload for the http request? We will see that in next section.

**Note**: HTTP trigger must have a payload, otherwise it is pretty useless to send a request without event data.

## Payload Construction
The http trigger payload has the following structure,

```yaml
payload:
  - src:
      dependencyName: test-dep
      dataKey: s3.bucket.name
    dest: bucket
  - src:
      dependencyName: test-dep
      contextKey: type
    dest: type
```

This looks very similar to the parameter structure you have seen in previous sections for trigger parameterization.

The `src` is the source of event. It contains,

  1. `dependencyName`: name of the event dependency to extract the event from.
  2. `dataKey`: to extract a particular key-value from event's data.
  3. `contextKey`: to extract a particular key-value from event' context.

The `dest` is the destination key within the result payload.

So, the above trigger payload will generate a request payload as,

```json
  {
    "bucket": "value_of_the_bucket_name_extracted_from_event_data",
    "type": "value_of_the_event_type_extracted_from_event_context"
  }
```

**_Note_**: If you define both the `contextKey` and `dataKey` within a payload item, then
the `dataKey` takes the precedence.

You can create any payload structure you want. To get more info on how to 
generate complex event payloads, take a look at [this library](https://github.com/tidwall/sjson).

The complete specification of HTTP trigger is available [here](https://github.com/argoproj/argo-events/blob/master/api/sensor.md#httptrigger).

## Serverless Workloads Integration
HTTP trigger provides an easy integration into 20+ event sources for the existing serverless workloads.

In this section, we will look at how to invoke a `Kubeless` function.

### Prerequisite
Kubeless must be installed :). You can follow the installation [here](https://kubeless.io/docs/quick-start/).
Make sure to deploy the `test` function.

### Invoke Function
1. First, lets create an http trigger for kubeless function,

   ```bash
    kubeless trigger http create hello --function-name hello
   ```
2. Update the sensor and update the `serverURL` to point to your Kubeless function URL.

3. Drop a file onto `test` bucket.

4. You will see the `hello` function getting invoked with following output,

   ```
   {'event-time': None, 'extensions': {'request': <LocalRequest: POST http://<URL>:8080/>}, 'event-type': None, 'event-namespace': None, 'data': '{"type":"minio","bucket":"test"}', 'event-id': None}
   ```

Note: The output was taken from `Kubeless` deployed on GCP.

## Policy
To determine whether the request was successful or not, HTTP trigger provides a `Status` policy.
The `Status` holds a list of response statuses that are considered valid.

1. Lets update the sensor with acceptable response statuses as [`200`, `300`] and point the http trigger to an invalid server url.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/09-http-trigger/sensor-02.yaml
   ```    

2. Drop a file onto `test` bucket.

3. Inspect the sensor logs, you will see the trigger resulted in failure.

   ```
   ERRO[2020-01-13 21:27:04] failed to operate on the event notification   error="policy application resulted in failure. http response status 404 is not allowed"
   ```

## Parameterization
You can either use `parameters` within http trigger or `parameter` under the template to parameterize
the trigger on fly.
