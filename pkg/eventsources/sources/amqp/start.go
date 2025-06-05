/*
Copyright 2018 The Argoproj Authors.

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
	"fmt"
	"time"

	"sigs.k8s.io/yaml"

	amqplib "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for amqp event source
type EventListener struct {
	EventSourceName string
	EventName       string
	AMQPEventSource aev1.AMQPEventSource
	Metrics         *metrics.Metrics
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
func (el *EventListener) GetEventSourceType() aev1.EventSourceType {
	return aev1.AMQPEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())

	log.Info("started processing the AMQP event source...")
	defer sources.Recover(el.GetEventName())

	amqpEventSource := &el.AMQPEventSource
	var conn *amqplib.Connection
	if err := sharedutil.DoWithRetry(amqpEventSource.ConnectionBackoff, func() error {
		c := amqplib.Config{
			Heartbeat: 10 * time.Second,
			Locale:    "en_US",
		}
		if amqpEventSource.TLS != nil {
			tlsConfig, err := sharedutil.GetTLSConfig(amqpEventSource.TLS)
			if err != nil {
				return fmt.Errorf("failed to get the tls configuration, %w", err)
			}
			c.TLSClientConfig = tlsConfig
		}
		if amqpEventSource.Auth != nil {
			username, err := sharedutil.GetSecretFromVolume(amqpEventSource.Auth.Username)
			if err != nil {
				return fmt.Errorf("username not found, %w", err)
			}
			password, err := sharedutil.GetSecretFromVolume(amqpEventSource.Auth.Password)
			if err != nil {
				return fmt.Errorf("password not found, %w", err)
			}
			c.SASL = []amqplib.Authentication{&amqplib.PlainAuth{
				Username: username,
				Password: password,
			}}
		}
		var err error
		var url string
		if amqpEventSource.URLSecret != nil {
			url, err = sharedutil.GetSecretFromVolume(amqpEventSource.URLSecret)
			if err != nil {
				return fmt.Errorf("urlSecret not found, %w", err)
			}
		} else {
			url = amqpEventSource.URL
		}
		conn, err = amqplib.DialConfig(url, c)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to connect to amqp broker for the event source %s, %w", el.GetEventName(), err)
	}

	log.Info("opening the server channel...")
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open the channel for the event source %s, %w", el.GetEventName(), err)
	}

	log.Info("checking parameters and set defaults...")
	setDefaults(amqpEventSource)

	log.Info("setting up the delivery channel...")
	delivery, err := getDelivery(ch, amqpEventSource)
	if err != nil {
		return fmt.Errorf("failed to get the delivery for the event source %s, %w", el.GetEventName(), err)
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
				return fmt.Errorf("channel might have been closed")
			}
			if err := el.handleOne(amqpEventSource, msg, dispatch, log); err != nil {
				log.Errorw("failed to process an AMQP message", zap.Error(err))
				el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			}
		case <-ctx.Done():
			err = conn.Close()
			if err != nil {
				log.Errorw("failed to close connection", zap.Error(err))
			}
			return nil
		}
	}
}

func (el *EventListener) handleOne(amqpEventSource *aev1.AMQPEventSource, msg amqplib.Delivery, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	log.Infow("received the message", zap.Any("message-id", msg.MessageId))
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
		return fmt.Errorf("failed to marshal the message, message-id: %s, %w", msg.MessageId, err)
	}

	log.Info("dispatching event ...")
	if err = dispatch(bodyBytes); err != nil {
		return fmt.Errorf("failed to dispatch AMQP event, %w", err)
	}
	return nil
}

// setDefaults sets the default values in case the user hasn't defined them
// helps also to keep retro-compatibility with current deployments
func setDefaults(eventSource *aev1.AMQPEventSource) {
	if eventSource.ExchangeDeclare == nil {
		eventSource.ExchangeDeclare = &aev1.AMQPExchangeDeclareConfig{
			Durable:    true,
			AutoDelete: false,
			Internal:   false,
			NoWait:     false,
		}
	}

	if eventSource.QueueDeclare == nil {
		eventSource.QueueDeclare = &aev1.AMQPQueueDeclareConfig{
			Name:       "",
			Durable:    false,
			AutoDelete: false,
			Exclusive:  true,
			NoWait:     false,
		}
	}

	if eventSource.QueueBind == nil {
		eventSource.QueueBind = &aev1.AMQPQueueBindConfig{
			NoWait: false,
		}
	}

	if eventSource.Consume == nil {
		eventSource.Consume = &aev1.AMQPConsumeConfig{
			ConsumerTag: "",
			AutoAck:     true,
			Exclusive:   false,
			NoLocal:     false,
			NoWait:      false,
		}
	}
}

// getDelivery sets up a channel for message deliveries
func getDelivery(ch *amqplib.Channel, eventSource *aev1.AMQPEventSource) (<-chan amqplib.Delivery, error) {
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
		return nil, fmt.Errorf("failed to declare exchange with name %s and type %s. err: %w", eventSource.ExchangeName, eventSource.ExchangeType, err)
	}
	optionalArguments, err := parseYamlTable(eventSource.QueueDeclare.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse optional queue declare table arguments from Yaml string: %w", err)
	}

	q, err := ch.QueueDeclare(
		eventSource.QueueDeclare.Name,
		eventSource.QueueDeclare.Durable,
		eventSource.QueueDeclare.AutoDelete,
		eventSource.QueueDeclare.Exclusive,
		eventSource.QueueDeclare.NoWait,
		optionalArguments,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = ch.QueueBind(
		q.Name,
		eventSource.RoutingKey,
		eventSource.ExchangeName,
		eventSource.QueueBind.NoWait,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind %s exchange '%s' to queue with routingKey: %s: %w", eventSource.ExchangeType, eventSource.ExchangeName, eventSource.RoutingKey, err)
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
		return nil, fmt.Errorf("failed to begin consuming messages: %w", err)
	}
	return delivery, nil
}

func parseYamlTable(argString string) (amqplib.Table, error) {
	if argString == "" {
		return nil, nil
	}
	var table amqplib.Table
	args := []byte(argString)
	err := yaml.Unmarshal(args, &table)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling Yaml to Table type. Args: %s. Err: %w", argString, err)
	}
	return table, nil
}
