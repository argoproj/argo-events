package main

// amqp contains configuration required to connect to rabbitmq service and process messages
// +k8s:openapi-gen=true
type amqp struct {
	// URL for rabbitmq service
	URL string `json:"url"`

	// ExchangeName is the exchange name
	// For more information, visit https://www.rabbitmq.com/tutorials/amqp-concepts.html
	ExchangeName string `json:"exchangeName"`

	// ExchangeType is rabbitmq exchange type
	ExchangeType string `json:"exchangeType"`

	// Routing key for bindings
	RoutingKey string `json:"routingKey"`
}
