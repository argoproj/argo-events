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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	amqplib "github.com/streadway/amqp"
	"k8s.io/apimachinery/pkg/util/wait"
)

// StartEventSource starts an event source
func (ese *AMQPEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)

	log.Info("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*amqp), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, ese.Log)
}

func getDelivery(ch *amqplib.Channel, a *amqp) (<-chan amqplib.Delivery, error) {
	err := ch.ExchangeDeclare(a.ExchangeName, a.ExchangeType, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange with name %s and type %s. err: %+v", a.ExchangeName, a.ExchangeType, err)
	}

	q, err := ch.QueueDeclare("", false, false, true, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %s", err)
	}

	err = ch.QueueBind(q.Name, a.RoutingKey, a.ExchangeName, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %s", a.ExchangeType, a.ExchangeName, a.RoutingKey, err)
	}

	delivery, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin consuming messages: %s", err)
	}
	return delivery, nil
}

func (ese *AMQPEventSourceExecutor) listenEvents(a *amqp, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	if err := gateways.Connect(&wait.Backoff{
		Steps:    a.Backoff.Steps,
		Factor:   a.Backoff.Factor,
		Duration: a.Backoff.Duration,
		Jitter:   a.Backoff.Jitter,
	}, func() error {
		var err error
		a.conn, err = amqplib.Dial(a.URL)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		errorCh <- err
		return
	}

	ch, err := a.conn.Channel()
	if err != nil {
		errorCh <- err
		return
	}

	delivery, err := getDelivery(ch, a)
	if err != nil {
		errorCh <- err
		return
	}

	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)

	log.Info("starting to subscribe to messages")
	for {
		select {
		case msg := <-delivery:
			dataCh <- msg.Body
		case <-doneCh:
			err = a.conn.Close()
			if err != nil {
				log.WithError(err).Info("failed to close connection")
			}
			return
		}
	}
}
