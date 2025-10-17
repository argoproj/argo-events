# Trigger Conditions

In the previous sections, you have been dealing with just a single dependency.
But, in many cases, you want to wait for multiple events to occur and then
trigger a resource which means you need a mechanism to determine which triggers
to execute based on set of different event dependencies. This mechanism is
supported through `conditions`.

**Note**: Whenever you define multiple dependencies in a sensor, the sensor
applies a `AND` operation, meaning, it will wait for all dependencies to resolve
before it executes triggers. `conditions` can modify that behavior.

## Prerequisite

Minio server must be set up in the `argo-events` namespace with a bucket called
`test` and it should be available at `minio-service.argo-events:9000`.

## Conditions

Consider a scenario where you have a `Webhook` and `Minio` event-source, and you
want to trigger an Argo workflow if the sensor receives an event from the
`Webhook` event-source, but, another workflow if it receives an event from the
`Minio` event-source.

1. Create the webhook event-source and event-source. The event-source listens
    to HTTP requests on port `12000`.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/06-trigger-conditions/webhook-event-source.yaml

2. Create the minio event-source. The event-source listens to events of type
    `PUT` and `DELETE` for objects in bucket `test`.

        kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/06-trigger-conditions/minio-event-source.yaml

Make sure there are no errors in any of the event-sources.

3. Let's create the sensor. If you take a closer look at the trigger templates,
    you will notice that it contains a field named `conditions`, which is a
    boolean expression containing dependency names. So, as soon as the expression
    is resolved as true, the corresponding trigger will be executed.

         kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/06-trigger-conditions/sensor-01.yaml

4. Send an HTTP request to Webhook event-source.

        curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example

5. You will notice an Argo workflow with name `group-1-xxxx` is created with
    following output,


        this is my first webhook


6. Now, lets generate a Minio event so that we can run `group-2-xxxx` workflow.
    Drop a file onto `test` bucket. The workflow that will get created will
    print the name of the bucket as follows,


        test
         
5. Great!! You have now learned how to use `conditions`. Lets update the sensor
    with a trigger that waits for both dependencies to resolve. This is the
    normal sensor behavior if `conditions` is not defined.

         kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/stable/examples/tutorials/06-trigger-conditions/sensor-02.yaml

    Send an HTTP request and perform a file drop on Minio bucket as done above.
    You should get the following output.

        
        this is my first webhook test
         
