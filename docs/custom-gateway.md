# Custom Gateway

You can write a gateway in 4 different ways,

1. Core gateway style.
2. Implement gateway as a gRPC server.
3. Implement gateway as a HTTP server.
4. Write your own implementation completely independent of framework.

Difference between first three options and fourth is that in options 1,2 and 3, framework provides a mechanism
to watch configuration updates, start/stop a configuration dynamically. In option 4, its up to
user to watch configuration updates and take actions.

<b> Below are the environment variables provided to all types of gateways </b>
 
 |  Field               |  Description |
 |----------------------|--------------|
 | TRANSFORMER_PORT     | http server port running within gateway transformer |
 |  GATEWAY_NAMESPACE | namespace of the gateway |
 |  GATEWAY_PROCESSOR_CONFIG_MAP            | map containing configurations to run in a gateway|
 |  GATEWAY_NAME             | name of the gateway |
 |  GATEWAY_CONTROLLER_INSTANCE_ID             | gateway controller instance id |
 
## Core Gateway Style
The most straightforward option. The gateway consists of two components,

1. Gateway Processor: either generates events internally or listens for external events and then 
passes those events to gateway-transformer

2. Gateway Transformer: transforms incoming events into cloudevents specification compliant events 
and dispatches them to watchers. 

![](core-gateway-style.png)
 
User needs to implement following interface.

```go
type ConfigExecutor interface {
	StartConfig(configContext *ConfigContext)
	StopConfig(configContext *ConfigContext)
	Validate(configContext *ConfigContext) error
}
```

ConfigData contains configuration key/name, value and the state of the configuration 
```go
type ConfigData struct {
	// Data holds the actual configuration
	Data *ConfigData

	StartChan chan struct{}
		
	StopChan chan struct{}

	DataChan chan []byte

	DoneChan chan struct{}

	ErrChan chan error

	ShutdownChan chan struct{}

	// Active tracks configuration state as running or stopped
	Active bool
	// Cancel is called to cancel the context used by client to communicate with gRPC server.
	// Use it only if gateway is implemented as gRPC server.
	Cancel context.CancelFunc
}
```

GatewayConfig contains generic configuration for a gateway
```go
type GatewayConfig struct {
	// Log provides fast and simple logger dedicated to JSON output
	Log zerolog.Logger
	// Clientset is client for kubernetes API
	Clientset kubernetes.Interface
	// Name is gateway name
	Name string
	// Namespace is namespace for the gateway to run inside
	Namespace string
	// KubeConfig rest client config
	KubeConfig *rest.Config	
}
```

* To send events back to framework for further processing, use
```go
gatewayConfig.DispatchEvent(event []byte, src string) error
```

For detailed implementation, check out [Core Gateways](https://github.com/argoproj/argo-events/tree/master/gateways/core)

## gRPC gateway
A gRPC gateway has 3 components, 
1.  Gateway Processor Server - your implementation of gRPC streaming server, either generates events or listens to 
external events and streams them back to gateway-processor-client

2. Gateway Processor Client - gRPC client provided by framework that connects to gateway processor server.

3. Gateway Transformer: transforms incoming events into cloudevents specification compliant events 
   and dispatches them to interested watchers. 

### Architecture
 ![](grpc-gateway.png)
 
To implement gateway processor server, you will need to implement
```proto
RunGateway(GatewayConfig) returns (stream Event)
```
`RunGateway` method takes a gateway configuration and sends events over a stream.

The gateway processor client opens a new connection for each gateway configuration and starts listening to
events on a stream.

For detailed implementation, check out [Calendar gRPC gateway](https://github.com/argoproj/argo-events/tree/master/gateways/grpc/calendar)

* To run gRPC gateway, you need to provide `rpcPort` in gateway spec.

* To write gateway gRPC server using other languages than go, generate server interfaces using protoc.
Follow protobuf tutorials []()https://developers.google.com/protocol-buffers/docs/tutorials

## HTTP Gateway
A gRPC gateway has 3 components, 
1.  Gateway Processor Server - your implementation of HTTP streaming server, either generates events or listens to 
external events and streams them back to gateway-processor-client. User code must accept POST requests on `/start` and `/stop`
endpoints.

2. Gateway Processor Client - sends configuration to gateway-processor-server on either `/start` endpoint for
a new configuration or `/stop` endpoint to stop a configuration. Processor client itself has a HTTP server 
running internally listening for events from gateway-processor-server.

3. Gateway Transformer: transforms incoming events into cloudevents specification compliant events 
   and dispatches them to watchers. 


### Architecture
![](http-gateway.png)

List of environment variables available to user code

|  Field               |  Description |
|----------------------|--------------|
| GATEWAY_PROCESSOR_SERVER_HTTP_PORT     | Gateway processor server HTTP server port |
|  GATEWAY_PROCESSOR_CLIENT_HTTP_PORT | Gateway processor client HTTP server port  |
|  GATEWAY_HTTP_CONFIG_START            | Endpoint to post new configuration |
|  GATEWAY_HTTP_CONFIG_STOP             | Endpoint to make configuration to stop |
|  GATEWAY_HTTP_CONFIG_EVENT             | Endpoint to send events to |
| GATEWAY_HTTP_CONFIG_RUNNING         | Endpoint to send activation notifications for the configuration /
| GATEWAY_HTTP_CONFIG_ERROR           | Endpoint on which which gateway processor listens for errors from configurations |

For detailed implementation, check out [Calendar HTTP gateway](https://github.com/argoproj/argo-events/tree/master/gateways/rest/calendar)

## Framework independent
The fourth option is you provide gateway implementation from scratch: watch the configuration
updates,  start/stop configuration if needed. Only requirement is that events must be 
dispatched to gateway-transformer using HTTP post request. The port to dispatch the event
is made available through environment variable `TRANSFORMER_PORT`.

### Gateway Examples
* Example gateway definitions are available at [here](https://github.com/argoproj/argo-events/tree/master/examples/gateways)