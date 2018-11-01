package webhook

// Webhook is a general purpose REST API
// +k8s:openapi-gen=true
type Webhook struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`

	// Port on which HTTP server is listening for incoming events.
	Port string `json:"port" protobuf:"bytes,3,opt,name=port"`
}
