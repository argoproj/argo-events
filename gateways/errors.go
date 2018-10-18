package gateways

const (
	ErrGatewayTransformerConnection = "failed to connect to gateway transformer"
	ErrGatewayEventWatch = "failed to watch k8 events for gateway configuration state updates"
	ErrGatewayConfigmapWatch = "failed to watch gateway configuration updates"
)