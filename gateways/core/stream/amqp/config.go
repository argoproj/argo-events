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
	"github.com/argoproj/argo-events/common"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	amqplib "github.com/streadway/amqp"
)

const ArgoEventsEventSourceVersion = "v0.10"

// AMQPEventSourceExecutor implements Eventing
type AMQPEventSourceExecutor struct {
	Log *logrus.Logger
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
	Backoff *common.Backoff `json:"backoff,omitempty"`
	// Connection manages the serialization and deserialization of frames from IO
	// and dispatches the frames to the appropriate channel.
	conn *amqplib.Connection
	// Maximum number of events consumed from the queue per RatePeriod.
	RateLimit uint32 `json:"rateLimit,omitempty"`
	// Number of seconds between two consumptions.
	RatePeriod uint32 `json:"ratePeriod,omitempty"`
}

func parseEventSource(eventSource string) (interface{}, error) {
	var a *amqp
	err := yaml.Unmarshal([]byte(eventSource), &a)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// Validate validates amqp
func (a *amqp) Validate() error {
	if (a.RateLimit == 0) != (a.RatePeriod == 0) {
		return errors.New("RateLimit and RatePeriod must be either set or omitted")
	}
	return nil
}
