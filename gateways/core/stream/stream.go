package stream

// Stream describes a queue stream resource
type Stream struct {
	// Type of the stream resource
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`
	// URL is the exposed endpoint for client connections to this service
	URL string `json:"url" protobuf:"bytes,2,opt,name=url"`
	// Attributes contains additional fields specific to each service implementation
	Attributes map[string]string `json:"attributes,omitempty" protobuf:"bytes,3,rep,name=attributes"`
}
