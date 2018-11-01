package main

// mqtt contains information to connect to MQTT broker
// +k8s:openapi-gen=true
type mqtt struct {
	// URL to connect to broker
	URL string `json:"url"`

	// Topic name
	Topic string `json:"topic"`

	// Client ID
	ClientId string
}
