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
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	amqplib "github.com/streadway/amqp"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for amqp event source
type EventListener struct {
	EventSourceName string
	EventName       string
	AMQPEventSource v1alpha1.AMQPEventSource
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.AMQPEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName()).Desugar()

	log.Info("started processing the AMQP event source...")
	defer sources.Recover(el.GetEventName())

	amqpEventSource := &el.AMQPEventSource
	backoff := common.GetConnectionBackoff(amqpEventSource.ConnectionBackoff)
	var conn *amqplib.Connection
	if err := common.Connect(backoff, func() error {
		if amqpEventSource.TLS != nil {
			tlsConfig, err := common.GetTLSConfig(amqpEventSource.TLS)
			if err != nil {
				return errors.Wrap(err, "failed to get the tls configuration")
			}
			conn, err = amqplib.DialTLS(amqpEventSource.URL, tlsConfig)
			if err != nil {
				return err
			}
		} else {
			var err error
			conn, err = amqplib.Dial(amqpEventSource.URL)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to amqp broker for the event source %s", el.GetEventName())
	}

	log.Info("opening the server channel...")
	ch, err := conn.Channel()
	if err != nil {
		return errors.Wrapf(err, "failed to open the channel for the event source %s", el.GetEventName())
	}

	log.Info("checking parameters and set defaults...")
	setDefaults(amqpEventSource)

	log.Info("setting up the delivery channel...")
	delivery, err := getDelivery(ch, amqpEventSource)
	if err != nil {
		return errors.Wrapf(err, "failed to get the delivery for the event source %s", el.GetEventName())
	}

	if amqpEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Info("listening to messages on channel...")
	for {
		select {
		case msg, ok := <-delivery:
			if !ok {
				log.Error("failed to read a message, channel might have been closed")
				return errors.New("channel might have been closed")
			}
			log.Info("received the message", zap.Any("message-id", msg.MessageId))
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
				Metadata:        amqpEventSource.Metadata,
			}
			if amqpEventSource.JSONBody {
				body.Body = (*json.RawMessage)(&msg.Body)
			} else {
				body.Body = msg.Body
			}

			bodyBytes, err := json.Marshal(body)
			if err != nil {
				log.Error("failed to marshal the message", zap.Any("message-id", msg.MessageId), zap.Error(err))
				continue
			}

			log.Info("dispatching event ...")
			err = dispatch(bodyBytes)
			if err != nil {
				log.Error("failed to dispatch AMQP event", zap.Error(err))
			}
		case <-ctx.Done():
			err = conn.Close()
			if err != nil {
				log.Error("failed to close connection", zap.Error(err))
			}
			return nil
		}
	}
}

// setDefaults sets the default values in case the user hasn't defined them
// helps also to keep retro-compatibility with current dpeloyments
func setDefaults(eventSource *v1alpha1.AMQPEventSource) {
	if eventSource.ExchangeDeclare == nil {
		eventSource.ExchangeDeclare = &v1alpha1.AMQPExchangeDeclareConfig{
			Durable:    true,
			AutoDelete: false,
			Internal:   false,
			NoWait:     false,
		}
	}

	if eventSource.QueueDeclare == nil {
		eventSource.QueueDeclare = &v1alpha1.AMQPQueueDeclareConfig{
			Name:       "",
			Durable:    false,
			AutoDelete: false,
			Exclusive:  true,
			NoWait:     false,
		}
	}

	if eventSource.QueueBind == nil {
		eventSource.QueueBind = &v1alpha1.AMQPQueueBindConfig{
			NoWait: false,
		}
	}

	if eventSource.Consume == nil {
		eventSource.Consume = &v1alpha1.AMQPConsumeConfig{
			ConsumerTag: "",
			AutoAck:     true,
			Exclusive:   false,
			NoLocal:     false,
			NoWait:      false,
		}
	}
}

// getDelivery sets up a channel for message deliveries
func getDelivery(ch *amqplib.Channel, eventSource *v1alpha1.AMQPEventSource) (<-chan amqplib.Delivery, error) {
	err := ch.ExchangeDeclare(
		eventSource.ExchangeName,
		eventSource.ExchangeType,
		eventSource.ExchangeDeclare.Durable,
		eventSource.ExchangeDeclare.AutoDelete,
		eventSource.ExchangeDeclare.Internal,
		eventSource.ExchangeDeclare.NoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Errorf("failed to declare exchange with name %s and type %s. err: %+v", eventSource.ExchangeName, eventSource.ExchangeType, err)
	}

	q, err := ch.QueueDeclare(
		eventSource.QueueDeclare.Name,
		eventSource.QueueDeclare.Durable,
		eventSource.QueueDeclare.AutoDelete,
		eventSource.QueueDeclare.Exclusive,
		eventSource.QueueDeclare.NoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Errorf("failed to declare queue: %s", err)
	}

	err = ch.QueueBind(
		q.Name,
		eventSource.RoutingKey,
		eventSource.ExchangeName,
		eventSource.QueueBind.NoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %s", eventSource.ExchangeType, eventSource.ExchangeName, eventSource.RoutingKey, err)
	}

	delivery, err := ch.Consume(
		q.Name,
		eventSource.Consume.ConsumerTag,
		eventSource.Consume.AutoAck,
		eventSource.Consume.Exclusive,
		eventSource.Consume.NoLocal,
		eventSource.Consume.NoWait,
		nil,
	)
	if err != nil {
		return nil, errors.Errorf("failed to begin consuming messages: %s", err)
	}
	return delivery, nil
}
