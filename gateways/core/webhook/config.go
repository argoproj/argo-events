package webhook

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
)

// WebhookConfigExecutor implements ConfigExecutor
type WebhookConfigExecutor struct {
	*gateways.GatewayConfig
}

// webhook is a general purpose REST API
// +k8s:openapi-gen=true
type webhook struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`

	// Port on which HTTP server is listening for incoming events.
	Port string `json:"port" protobuf:"bytes,3,opt,name=port"`
}

func parseConfig(config string) (*webhook, error) {
	var n *webhook
	err := yaml.Unmarshal([]byte(config), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
