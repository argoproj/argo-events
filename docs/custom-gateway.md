# Custom Gateway

To implement a custom gateway, you need to create a gRPC server and implement the service defined below.
The framework code acts as a gRPC client consuming event stream from gateway server.

1. [Proto definition](#proto-definition)
2. [Available environment variables](#available-environment-variables-to-server)
3. [Implementation](#implementation)

<br/>

<p align="center">
  <img src="https://github.com/argoproj/argo-events/blob/update-docs/docs/custom-gateway.png?raw=true" alt="Sensor"/>
</p>

<br/>

## Proto Definition
1. The proto file is located at https://github.com/argoproj/argo-events/blob/master/gateways/eventing.proto 

2. If you choose to implement the gateway in `go`, then you can find generated client stubs at https://github.com/argoproj/argo-events/blob/master/gateways/eventing.pb.go

3. To create stubs in other languages, head over to gRPC website https://grpc.io/
 
4. Service
    ```proto
    /**
    * Service for handling event sources.
    */
    service Eventing {
        // StartEventSource starts an event source and returns stream of events.
        rpc StartEventSource(EventSource) returns (stream Event);
        // ValidateEventSource validates an event source.
        rpc ValidateEventSource(EventSource) returns (ValidEventSource);
    }
    ```

## Available Environment Variables to Server
 
 |  Field               |  Description |
 |----------------------|--------------|
 |  GATEWAY_NAMESPACE                           | K8s namespace of the gateway |
 |  GATEWAY_EVENT_SOURCE_CONFIG_MAP            | K8s configmap containing event source|
 |  GATEWAY_NAME                               | name of the gateway |
 |  GATEWAY_CONTROLLER_INSTANCE_ID             | gateway controller instance id |
 | GATEWAY_CONTROLLER_NAME                     | gateway controller name
 | GATEWAY_SERVER_PORT                         | Port on which the gateway gRPC server should run 
 
## Implementation
