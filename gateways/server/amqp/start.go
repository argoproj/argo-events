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
	"encoding/json"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	amqplib "github.com/streadway/amqp"
)

// EventListener implements Eventing for amqp event source
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
}

// Data refers to the event data.
type Data struct {
	ContentType     string    `json:"contentType"`     // MIME content type
	ContentEncoding string    `json:"contentEncoding"` // MIME content encoding
	DeliveryMode    int       `json:"deliveryMode"`    // queue implementation use - non-persistent (1) or persistent (2)
	Priority        int       `json:"priority"`        // queue implementation use - 0 to 9
	CorrelationId   string    `json:"correlationId"`   // application use - correlation identifier
	ReplyTo         string    `json:"replyTo"`         // application use - address to reply to (ex: RPC)
	Expiration      string    `json:"expiration"`      // implementation use - message expiration spec
	MessageId       string    `json:"messageId"`       // application use - message identifier
	Timestamp       time.Time `json:"timestamp"`       // application use - message timestamp
	Type            string    `json:"type"`            // application use - message type name
	AppId           string    `json:"appId"`           // application use - creating application id
	Exchange        string    `json:"exchange"`        // basic.publish exchange
	RoutingKey      string    `json:"routingKey"`      // basic.publish routing key
	Body            []byte    `json:"body"`
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
	backoff := common.GetConnectionBackoff(amqpEventSource.ConnectionBackoff)
	if err := server.Connect(backoff, func() error {
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
			logger.WithField("message-id", msg.MessageId).Infoln("received the message")
			body := &Data{
				ContentType:     msg.ContentType,
				ContentEncoding: msg.ContentEncoding,
				DeliveryMode:    int(msg.DeliveryMode),
				Priority:        int(msg.Priority),
				CorrelationId:   msg.CorrelationId,
				ReplyTo:         msg.ReplyTo,
				Expiration:      msg.Expiration,
				MessageId:       msg.MessageId,
				Timestamp:       msg.Timestamp,
				Type:            msg.Type,
				AppId:           msg.AppId,
				Exchange:        msg.Exchange,
				RoutingKey:      msg.RoutingKey,
				Body:            msg.Body,
			}

			bodyBytes, err := json.Marshal(body)
			if err != nil {
				logger.WithError(err).WithField("message-id", msg.MessageId).Errorln("failed to marshal the message")
				continue
			}

			logger.Infoln("dispatching event on data channel...")
			dataCh <- bodyBytes
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
