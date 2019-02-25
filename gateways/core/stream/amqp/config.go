/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package amqp

import (
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
	amqplib "github.com/streadway/amqp"
	"k8s.io/apimachinery/pkg/util/wait"
)

// AMQPEventSourceExecutor implements Eventing
type AMQPEventSourceExecutor struct {
	Log zerolog.Logger
}

// amqp contains configuration required to connect to rabbitmq service and process messages
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
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
	// Connection manages the serialization and deserialization of frames from IO
	// and dispatches the frames to the appropriate channel.
	conn *amqplib.Connection
}

func parseEventSource(eventSource string) (interface{}, error) {
	var a *amqp
	err := yaml.Unmarshal([]byte(eventSource), &a)
	if err != nil {
		return nil, err
	}
	return a, nil
}
