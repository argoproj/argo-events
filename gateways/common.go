package gateways

// Gateway constants
const (
	// LabelEventSourceName is the label for a event source in gateway
	LabelEventSourceName = "event-source-name"
	// LabelEventSourceID is the label for gateway configuration ID
	LabelEventSourceID      = "event-source-id"
	EnvVarGatewayServerPort = "GATEWAY_SERVER_PORT"
	// Server Connection Timeout, 10 seconds
	ServerConnTimeout = 10
	// EventSourceDir
	EventSourceDir = "../../../examples/eventsources"
)
