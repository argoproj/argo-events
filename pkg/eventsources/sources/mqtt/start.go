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

package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common/logging"
	metrics "github.com/argoproj/argo-events/metrics"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for mqtt event source
type EventListener struct {
	EventSourceName string
	EventName       string
	MQTTEventSource v1alpha1.MQTTEventSource
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
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.MQTTEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	log.Info("starting MQTT event source...")
	mqttEventSource := &el.MQTTEventSource

	if mqttEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Info("setting up the message handler...")
	handler := func(c mqttlib.Client, msg mqttlib.Message) {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		eventData := &events.MQTTEventData{
			Topic:     msg.Topic(),
			MessageID: int(msg.MessageID()),
			Body:      msg.Payload(),
			Metadata:  mqttEventSource.Metadata,
		}
		if mqttEventSource.JSONBody {
			body := msg.Payload()
			eventData.Body = (*json.RawMessage)(&body)
		} else {
			eventData.Body = msg.Payload()
		}

		eventBody, err := json.Marshal(eventData)
		if err != nil {
			log.Errorw("failed to marshal the event data, rejecting the event...", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			return
		}
		log.Info("dispatching event on the data channel...")
		if err = dispatch(eventBody); err != nil {
			log.Errorw("failed to dispatch MQTT event...", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		}
	}

	log.Info("setting up the mqtt broker client...")
	opts := mqttlib.NewClientOptions().AddBroker(mqttEventSource.URL).SetClientID(mqttEventSource.ClientID)
	if mqttEventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(mqttEventSource.TLS)
		if err != nil {
			return fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		opts.TLSConfig = tlsConfig
	}

	if mqttEventSource.Auth != nil {
		username, err := sharedutil.GetSecretFromVolume(mqttEventSource.Auth.Username)
		if err != nil {
			return fmt.Errorf("username not found, %w", err)
		}
		password, err := sharedutil.GetSecretFromVolume(mqttEventSource.Auth.Password)
		if err != nil {
			return fmt.Errorf("password not found, %w", err)
		}
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	var client mqttlib.Client

	log.Info("connecting to mqtt broker...")
	if err := sharedutil.DoWithRetry(mqttEventSource.ConnectionBackoff, func() error {
		client = mqttlib.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to connect to the mqtt broker for event source %s, %w", el.GetEventName(), err)
	}

	log.Info("subscribing to the topic...")
	if token := client.Subscribe(mqttEventSource.Topic, 0, handler); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to the topic %s for event source %s, %w", mqttEventSource.Topic, el.GetEventName(), token.Error())
	}

	<-ctx.Done()
	log.Info("event source is stopped, unsubscribing the client...")

	token := client.Unsubscribe(mqttEventSource.Topic)
	if token.Error() != nil {
		log.Errorw("failed to unsubscribe client", zap.Error(token.Error()))
	}

	return nil
}
