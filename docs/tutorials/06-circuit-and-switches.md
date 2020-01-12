# Circuit and Switches
In previous sections, you have been dealing with just a single dependency. But, in many
cases, you want to wait for multiple events to occur and then trigger a resource which means
you need a mechanism to determine which triggers to execute based on set of different event dependencies.
This mechanism is supported through `Circuit` and `Switches`.

<b>Note</b>: Whenever you define multiple dependencies in a sensor, the sensor applies
a `AND` operation, meaning, it will wait for all dependencies to resolve before it executes triggers.
`Circuit` and `Switches` can modify that behavior. 

## Prerequisite
Minio server must be set up in the `argo-events` namespace and it should be available
at `minio-service.argo-events:9000`.

## Circuit
A circuit is a boolean expression. To create a circuit, you just need to define event
dependencies in groups and the sensor will apply the circuit logic on those groups.
If the logic results in `true` value, the sensor will execute the triggers else it won't.

Consider a scenario where you have a `Webhook` and `Minio` gateway, and you want
to trigger an Argo workflow if the sensor receives event from the `Webhook` gateway,
but, another workflow if it receives event from the `Minio` gateway.

1. Create the webhook event source and gateway. The gateway listens to HTTP requests
   on port `12000`

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/05-circuit-and-switches/webhook-event-source.yaml
   ```

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/05-circuit-and-switches/webhook-gateway.yaml
   ```

2. Create the minio event source and gateway. The gateway listens to events of type
   `PUT` and `DELETE` for objects in bucket `input`.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/05-circuit-and-switches/minio-event-source.yaml
   ```

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/05-circuit-and-switches/minio-gateway.yaml
   ```

Make sure there are no errors in any of the gateways and all event sources are active.

3. Lets create the sensor. If you take a closer look at the trigger templates, you will
   notice that it contains `switch` key with `all` condition, meaning, execute this trigger
   when every group defined in `all` is resolved. In the sensor definition, there
   is only one group under `all` in both trigger templates. So, as soon as the group is resolved, the
   corresponding trigger will be executed.

   ```bash
   kubectl -n argo-events apply -f https://raw.githubusercontent.com/argoproj/argo-events/master/examples/tutorials/05-circuit-and-switches/sensor-01.yaml
   ```  
   
4. Send a HTTP request to Webhook gateway,

   ```bash
   curl -d '{"message":"this is my first webhook"}' -H "Content-Type: application/json" -X POST http://localhost:12000/example
   ```

5. You will notice an Argo worklfow with name `group-1-xxxx` is created with following output,

   
