package webhook

import (
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
	"net/http"
)

// WebhookEventSourceExecutor implements Eventing
type WebhookEventSourceExecutor struct {
	Log zerolog.Logger
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

	// srv holds reference to http server
	// +k8s:openapi-gen=false
	srv *http.Server `json:"srv,omitempty"`
	// +k8s:openapi-gen=false
	mux *http.ServeMux `json:"mux,omitempty"`
}

func parseEventSource(es *string) (*webhook, error) {
	var n *webhook
	err := yaml.Unmarshal([]byte(*es), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
