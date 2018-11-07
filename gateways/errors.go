package gateways

import (
	"errors"
)

const (
	ErrGatewayTransformerConnectionMsg = "failed to connect to gateway transformer"
	ErrGatewayEventWatchMsg            = "failed to watch k8 events for gateway configuration state updates"
	ErrGatewayConfigmapWatchMsg        = "failed to watch gateway configuration updates"
)

var (
	ErrConfigParseFailed = errors.New("failed to parse configuration")
	ErrEmptyConfig = errors.New("configuration must be non empty")
	ErrInvalidConfig     = errors.New("invalid configuration")
)
