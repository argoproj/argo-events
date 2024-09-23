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

package emitter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	emitter "github.com/emitter-io/go/v2"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for Emitter event source
type EventListener struct {
	EventSourceName    string
	EventName          string
	EmitterEventSource v1alpha1.EmitterEventSource
	Metrics            *metrics.Metrics
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
	return v1alpha1.EmitterEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the Emitter event source...")
	defer sources.Recover(el.GetEventName())

	emitterEventSource := &el.EmitterEventSource

	var options []func(client *emitter.Client)
	if emitterEventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(emitterEventSource.TLS)
		if err != nil {
			return fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		options = append(options, emitter.WithTLSConfig(tlsConfig))
	}
	options = append(options, emitter.WithBrokers(emitterEventSource.Broker), emitter.WithAutoReconnect(true))

	if emitterEventSource.Username != nil {
		username, err := sharedutil.GetSecretFromVolume(emitterEventSource.Username)
		if err != nil {
			return fmt.Errorf("failed to retrieve the username from %s, %w", emitterEventSource.Username.Name, err)
		}
		options = append(options, emitter.WithUsername(username))
	}

	if emitterEventSource.Password != nil {
		password, err := sharedutil.GetSecretFromVolume(emitterEventSource.Password)
		if err != nil {
			return fmt.Errorf("failed to retrieve the password from %s, %w", emitterEventSource.Password.Name, err)
		}
		options = append(options, emitter.WithPassword(password))
	}

	if emitterEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Infow("creating a client", zap.Any("channelName", emitterEventSource.ChannelName))
	client := emitter.NewClient(options...)

	if err := sharedutil.DoWithRetry(emitterEventSource.ConnectionBackoff, func() error {
		if err := client.Connect(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to connect to %s, %w", emitterEventSource.Broker, err)
	}

	if err := client.Subscribe(emitterEventSource.ChannelKey, emitterEventSource.ChannelName, func(_ *emitter.Client, message emitter.Message) {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		body := message.Payload()
		event := &events.EmitterEventData{
			Topic:    message.Topic(),
			Body:     body,
			Metadata: emitterEventSource.Metadata,
		}
		if emitterEventSource.JSONBody {
			event.Body = (*json.RawMessage)(&body)
		}
		eventBytes, err := json.Marshal(event)

		if err != nil {
			log.Errorw("failed to marshal the event data", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			return
		}
		log.Info("dispatching event on data channel...")
		if err = dispatch(eventBytes); err != nil {
			log.Errorw("failed to dispatch event", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		}
	}); err != nil {
		return fmt.Errorf("failed to subscribe to channel %s, %w", emitterEventSource.ChannelName, err)
	}

	<-ctx.Done()

	log.Infow("event source stopped, unsubscribe the channel", zap.Any("channelName", emitterEventSource.ChannelName))

	if err := client.Unsubscribe(emitterEventSource.ChannelKey, emitterEventSource.ChannelName); err != nil {
		log.Errorw("failed to unsubscribe", zap.Any("channelName", emitterEventSource.ChannelName), zap.Error(err))
	}

	return nil
}
