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
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	amqplib "github.com/streadway/amqp"
	"k8s.io/apimachinery/pkg/util/wait"
)

// EventListener implements Eventing for amqp event source
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)

	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listens to events from amqp server
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var amqpEventSource *v1alpha1.AMQPEventSource
	if err := yaml.Unmarshal(eventSource.Value, &amqpEventSource); err != nil {
		errorCh <- err
		return
	}

	var conn *amqplib.Connection

	logger.Infoln("dialing connection...")
	if err := server.Connect(&wait.Backoff{
		Steps:    amqpEventSource.ConnectionBackoff.Steps,
		Factor:   amqpEventSource.ConnectionBackoff.Factor,
		Duration: amqpEventSource.ConnectionBackoff.Duration,
		Jitter:   amqpEventSource.ConnectionBackoff.Jitter,
	}, func() error {
		var err error
		conn, err = amqplib.Dial(amqpEventSource.URL)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		errorCh <- err
		return
	}

	logger.Infoln("opening the server channel...")
	ch, err := conn.Channel()
	if err != nil {
		errorCh <- err
		return
	}

	logger.Infoln("setting up the delivery channel...")
	delivery, err := getDelivery(ch, amqpEventSource)
	if err != nil {
		errorCh <- err
		return
	}

	logger.Info("listening to messages on channel...")
	for {
		select {
		case msg := <-delivery:
			logger.Infoln("dispatching event on data channel...")
			dataCh <- msg.Body
		case <-doneCh:
			err = conn.Close()
			if err != nil {
				logger.WithError(err).Info("failed to close connection")
			}
			return
		}
	}
}

// getDelivery sets up a channel for message deliveries
func getDelivery(ch *amqplib.Channel, eventSource *v1alpha1.AMQPEventSource) (<-chan amqplib.Delivery, error) {
	err := ch.ExchangeDeclare(eventSource.ExchangeName, eventSource.ExchangeType, true, false, false, false, nil)
	if err != nil {
		return nil, errors.Errorf("failed to declare exchange with name %s and type %s. err: %+v", eventSource.ExchangeName, eventSource.ExchangeType, err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, errors.Errorf("failed to declare queue: %s", err)
	}

	err = ch.QueueBind(q.Name, eventSource.RoutingKey, eventSource.ExchangeName, false, nil)
	if err != nil {
		return nil, errors.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %s", eventSource.ExchangeType, eventSource.ExchangeName, eventSource.RoutingKey, err)
	}

	delivery, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, errors.Errorf("failed to begin consuming messages: %s", err)
	}
	return delivery, nil
}
