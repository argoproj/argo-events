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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")
	channels := server.NewChannels()

	go server.HandleEventsFromEventSource(eventSource.Name, eventStream, channels, listener.Logger)

	defer func() {
		channels.Stop <- struct{}{}
	}()

	if err := listener.listenEvents(eventSource, channels); err != nil {
		listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithError(err).Errorln("failed to listen to events")
		return err
	}

	return nil
}

// listenEvents listens to events from amqp server
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var amqpEventSource *v1alpha1.AMQPEventSource
	if err := yaml.Unmarshal(eventSource.Value, &amqpEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	logger.Infoln("dialing connection...")
	backoff := common.GetConnectionBackoff(amqpEventSource.ConnectionBackoff)
	var conn *amqplib.Connection
	if err := server.Connect(backoff, func() error {
		var err error
		if amqpEventSource.TLS != nil {
			tlsConfig, err := common.GetTLSConfig(amqpEventSource.TLS.CACertPath, amqpEventSource.TLS.ClientCertPath, amqpEventSource.TLS.ClientKeyPath)
			if err != nil {
				return errors.Wrap(err, "failed to get the tls configuration")
			}
			conn, err = amqplib.DialTLS(amqpEventSource.URL, tlsConfig)
			if err != nil {
				return err
			}
		} else {
			conn, err = amqplib.Dial(amqpEventSource.URL)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to amqp broker for the event source %s", eventSource.Name)
	}

	logger.Infoln("opening the server channel...")
	ch, err := conn.Channel()
	if err != nil {
		return errors.Wrapf(err, "failed to open the channel for the event source %s", eventSource.Name)
	}

	logger.Infoln("setting up the delivery channel...")
	delivery, err := getDelivery(ch, amqpEventSource)
	if err != nil {
		return errors.Wrapf(err, "failed to get the delivery for the event source %s", eventSource.Name)
	}

	if amqpEventSource.JSONBody {
		logger.Infoln("assuming all events have a json body...")
	}

	logger.Info("listening to messages on channel...")
	for {
		select {
		case msg := <-delivery:
			logger.WithField("message-id", msg.MessageId).Infoln("received the message")
			body := &events.AMQPEventData{
				ContentType:     msg.ContentType,
				ContentEncoding: msg.ContentEncoding,
				DeliveryMode:    int(msg.DeliveryMode),
				Priority:        int(msg.Priority),
				CorrelationId:   msg.CorrelationId,
				ReplyTo:         msg.ReplyTo,
				Expiration:      msg.Expiration,
				MessageId:       msg.MessageId,
				Timestamp:       msg.Timestamp.String(),
				Type:            msg.Type,
				AppId:           msg.AppId,
				Exchange:        msg.Exchange,
				RoutingKey:      msg.RoutingKey,
			}
			if amqpEventSource.JSONBody {
				body.Body = (*json.RawMessage)(&msg.Body)
			} else {
				body.Body = msg.Body
			}

			bodyBytes, err := json.Marshal(body)
			if err != nil {
				logger.WithError(err).WithField("message-id", msg.MessageId).Errorln("failed to marshal the message")
				continue
			}

			logger.Infoln("dispatching event on data channel...")
			channels.Data <- bodyBytes
		case <-channels.Done:
			err = conn.Close()
			if err != nil {
				logger.WithError(err).Info("failed to close connection")
			}
			return nil
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
