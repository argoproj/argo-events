package main

// nats contains configuration to connect to NATS cluster
// +k8s:openapi-gen=true
type nats struct {
	// URL to connect to nats cluster
	URL string `json:"url"`

	// Subject name
	Subject string `json:"subject"`
}
