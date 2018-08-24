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
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	log "github.com/sirupsen/logrus"
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

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
type amqp struct{}

// New creates an amqp listener
func New() sdk.Listener {
	return new(amqp)
}

func (*amqp) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	conn, err := amqplib.Dial(signal.Stream.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %s", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %s", err)
	}

	delivery, err := getDelivery(ch, signal.Stream.Attributes)
	events := make(chan *v1alpha1.Event)

	// start listening for messages
	go func() {
		defer close(events)
		for {
			select {
			case msg := <-delivery:
				event := &v1alpha1.Event{
					Context: v1alpha1.EventContext{
						EventID:            msg.MessageId,
						EventType:          EventType,
						EventTypeVersion:   msg.ConsumerTag,
						CloudEventsVersion: sdk.CloudEventsVersion,
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
			case <-done:
				if err := ch.Close(); err != nil {
					log.Printf("failed to close channel for signal '%s': %s", signal.Name, err)
				}
				if err := conn.Close(); err != nil {
					log.Panicf("failed to close connection for signal '%s': %s", signal.Name, err)
				}
				log.Printf("shut down signal '%s'", signal.Name)
				return
			}
		}
	}()

	return events, nil
}

func getDelivery(ch *amqplib.Channel, attr map[string]string) (<-chan amqplib.Delivery, error) {
	exName, exType, rKey, err := parseAttributes(attr)
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(exName, exType, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare %s exchange '%s': %s", exType, exName, err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %s", err)
	}

	err = ch.QueueBind(q.Name, rKey, exName, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %s", exType, exName, rKey, err)
	}

	delivery, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin consuming messages: %s", err)
	}
	return delivery, nil
}

func parseAttributes(attr map[string]string) (string, string, string, error) {
	// parse out the attributes
	exchangeName, ok := attr[exchangeNameKey]
	if !ok {
		return "", "", "", sdk.ErrMissingRequiredAttribute
	}
	exchangeType, ok := attr[exchangeTypeKey]
	if !ok {
		return exchangeName, "", "", sdk.ErrMissingRequiredAttribute
	}
	routingKey, ok := attr[routingKeyKey]
	if !ok {
		return exchangeName, "", "", sdk.ErrMissingRequiredAttribute
	}
	return exchangeName, exchangeType, routingKey, nil
}
