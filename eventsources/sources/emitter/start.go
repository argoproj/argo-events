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
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	log.Infoln("started processing the Emitter event source...")
	defer sources.Recover(el.GetEventName())

	emitterEventSource := &el.EmitterEventSource

	var options []func(client *emitter.Client)
	if emitterEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(emitterEventSource.TLS.CACertPath, emitterEventSource.TLS.ClientCertPath, emitterEventSource.TLS.ClientKeyPath)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		options = append(options, emitter.WithTLSConfig(tlsConfig))
	}
	options = append(options, emitter.WithBrokers(emitterEventSource.Broker), emitter.WithAutoReconnect(true))

	if emitterEventSource.Username != nil {
		username, ok := common.GetEnvFromSecret(emitterEventSource.Username)
		if !ok {
			return errors.Errorf("failed to retrieve the username from %s in ENV", emitterEventSource.Username.Name)
		}
		options = append(options, emitter.WithUsername(username))
	}

	if emitterEventSource.Password != nil {
		password, ok := common.GetEnvFromSecret(emitterEventSource.Password)
		if !ok {
			return errors.Errorf("failed to retrieve the password from %s in ENV", emitterEventSource.Password.Name)
		}
		options = append(options, emitter.WithPassword(password))
	}

	if emitterEventSource.JSONBody {
		log.Infoln("assuming all events have a json body...")
	}

	log.WithField("channel-name", emitterEventSource.ChannelName).Infoln("creating a client")
	client := emitter.NewClient(options...)

	if err := sources.Connect(common.GetConnectionBackoff(emitterEventSource.ConnectionBackoff), func() error {
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
			Topic: message.Topic(),
			Body:  body,
		}
		if emitterEventSource.JSONBody {
			event.Body = (*json.RawMessage)(&body)
		}
		eventBytes, err := json.Marshal(event)

		if err != nil {
			log.WithError(err).Errorln("failed to marshal the event data")
			return
		}
		log.Infoln("dispatching event on data channel...")
		err = dispatch(eventBytes)
		if err != nil {
			log.WithError(err).Errorln("failed to dispatch event")
		}
	}); err != nil {
		return errors.Wrapf(err, "failed to subscribe to channel %s", emitterEventSource.ChannelName)
	}

	<-stopCh

	log.WithField("channel-name", emitterEventSource.ChannelName).Infoln("event source stopped, unsubscribe the channel")

	if err := client.Unsubscribe(emitterEventSource.ChannelKey, emitterEventSource.ChannelName); err != nil {
		log.WithError(err).WithField("channel-name", emitterEventSource.ChannelName).Errorln("failed to unsubscribe")
	}

	return nil
}
