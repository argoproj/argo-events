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

package main

import (
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
	plugin "github.com/hashicorp/go-plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	amqplib "github.com/streadway/amqp"
)

const (
	// EventType defines the amqp event type
	EventType       = "amqp"
	exchangeNameKey = "exchangeName"
	exchangeTypeKey = "exchangeType"
	routingKeyKey   = "routingKey"
)

type amqp struct {
	conn     *amqplib.Connection
	delivery <-chan amqplib.Delivery
}

// New creates an amqp signaler
func New() shared.Signaler {
	return &amqp{}
}

func (a *amqp) Start(signal *v1alpha1.Signal) (<-chan *v1alpha1.Event, error) {
	// parse out the attributes
	exchangeName, ok := signal.Stream.Attributes[exchangeNameKey]
	if !ok {
		return nil, shared.ErrMissingRequiredAttribute
	}
	exchangeType, ok := signal.Stream.Attributes[exchangeTypeKey]
	if !ok {
		return nil, shared.ErrMissingRequiredAttribute
	}
	routingKey, ok := signal.Stream.Attributes[routingKeyKey]
	if !ok {
		return nil, shared.ErrMissingRequiredAttribute
	}

	var err error
	a.conn, err = amqplib.Dial(signal.Stream.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ. cause: %s", err)
	}

	ch, err := a.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel. cause: %s", err)
	}

	err = ch.ExchangeDeclare(exchangeName, exchangeType, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare RabbitMQ %s exchange '%s'. cause: %s", exchangeType, exchangeName, err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare RabbigMQ queue. cause: %s", err)
	}

	err = ch.QueueBind(q.Name, routingKey, exchangeName, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind RabbitMQ %s exchange '%s' to queue with routingKey: %s. cause: %s", exchangeType, exchangeName, routingKey, err)
	}
	a.delivery, err = ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin consuming RabbitMQ messages. cause: %s", err)
	}

	events := make(chan *v1alpha1.Event)
	go a.listen(events)
	return events, nil
}

func (a *amqp) Stop() error {
	return a.conn.Close()
}

func (a *amqp) listen(events chan *v1alpha1.Event) {
	for msg := range a.delivery {
		event := &v1alpha1.Event{
			Context: v1alpha1.EventContext{
				EventID:            msg.MessageId,
				EventType:          EventType,
				EventTypeVersion:   msg.ConsumerTag,
				CloudEventsVersion: shared.CloudEventsVersion,
				Source:             &v1alpha1.URI{},
				EventTime:          metav1.Time{Time: msg.Timestamp},
				SchemaURL:          &v1alpha1.URI{},
				ContentType:        msg.ContentType,
				Extensions:         map[string]string{"content-encoding": msg.ContentEncoding},
			},
			Data: msg.Body,
		}
		events <- event
		msg.Ack(false)
	}
}

func main() {
	amqp := New()
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			shared.SignalPluginName: shared.NewPlugin(amqp),
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
