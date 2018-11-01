package main

// kafka defines configuration required to connect to kafka cluster
// +k8s:openapi-gen=true
type kafka struct {
	// URL to kafka cluster
	URL string `json:"url"`

	// Partition name
	Partition string `json:"partition"`

	// Topic name
	Topic string `json:"topic"`
}
