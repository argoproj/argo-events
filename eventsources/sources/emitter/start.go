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

package emitter

import (
	"context"
	"encoding/json"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	emitter "github.com/emitter-io/go/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// EventListener implements Eventing for Emitter event source
type EventListener struct {
	EventSourceName    string
	EventName          string
	EmitterEventSource v1alpha1.EmitterEventSource
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
	return apicommon.EmitterEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName()).Desugar()
	log.Info("started processing the Emitter event source...")
	defer sources.Recover(el.GetEventName())

	emitterEventSource := &el.EmitterEventSource

	var options []func(client *emitter.Client)
	if emitterEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(emitterEventSource.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		options = append(options, emitter.WithTLSConfig(tlsConfig))
	}
	options = append(options, emitter.WithBrokers(emitterEventSource.Broker), emitter.WithAutoReconnect(true))

	if emitterEventSource.Username != nil {
		username, err := common.GetSecretFromVolume(emitterEventSource.Username)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve the username from %s", emitterEventSource.Username.Name)
		}
		options = append(options, emitter.WithUsername(username))
	}

	if emitterEventSource.Password != nil {
		password, err := common.GetSecretFromVolume(emitterEventSource.Password)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve the password from %s", emitterEventSource.Password.Name)
		}
		options = append(options, emitter.WithPassword(password))
	}

	if emitterEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Info("creating a client", zap.Any("channelName", emitterEventSource.ChannelName))
	client := emitter.NewClient(options...)

	if err := common.Connect(emitterEventSource.ConnectionBackoff, func() error {
		if err := client.Connect(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to %s", emitterEventSource.Broker)
	}

	if err := client.Subscribe(emitterEventSource.ChannelKey, emitterEventSource.ChannelName, func(_ *emitter.Client, message emitter.Message) {
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
			log.Error("failed to marshal the event data", zap.Error(err))
			return
		}
		log.Info("dispatching event on data channel...")
		err = dispatch(eventBytes)
		if err != nil {
			log.Error("failed to dispatch event", zap.Error(err))
		}
	}); err != nil {
		return errors.Wrapf(err, "failed to subscribe to channel %s", emitterEventSource.ChannelName)
	}

	<-ctx.Done()

	log.Info("event source stopped, unsubscribe the channel", zap.Any("channelName", emitterEventSource.ChannelName))

	if err := client.Unsubscribe(emitterEventSource.ChannelKey, emitterEventSource.ChannelName); err != nil {
		log.Error("failed to unsubscribe", zap.Any("channelName", emitterEventSource.ChannelName), zap.Error(err))
	}

	return nil
}
