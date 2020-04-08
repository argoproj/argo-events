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
	"encoding/json"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	emitter "github.com/emitter-io/go/v2"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for Emitter event source
type EventListener struct {
	// K8sClient is the kubernetes client
	K8sClient kubernetes.Interface
	// Logger to log stuff
	Logger    *logrus.Logger
	Namespace string
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

// listenEvents listens events from emitter subscription
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSourceName, eventSource.Name)

	logger.Infoln("parsing the event source")
	var emitterEventSource *v1alpha1.EmitterEventSource
	if err := yaml.Unmarshal(eventSource.Value, &emitterEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	if emitterEventSource.Namespace == "" {
		emitterEventSource.Namespace = listener.Namespace
	}

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
		username, err := common.GetSecrets(listener.K8sClient, emitterEventSource.Namespace, emitterEventSource.Username)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve the username from %s", emitterEventSource.Username.Name)
		}
		options = append(options, emitter.WithUsername(username))
	}

	if emitterEventSource.Password != nil {
		password, err := common.GetSecrets(listener.K8sClient, emitterEventSource.Namespace, emitterEventSource.Password)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve the password from %s", emitterEventSource.Password.Name)
		}
		options = append(options, emitter.WithPassword(password))
	}

	if emitterEventSource.JSONBody {
		logger.Infoln("assuming all events have a json body...")
	}

	logger.WithField("channel-name", emitterEventSource.ChannelName).Infoln("creating a client")
	client := emitter.NewClient(options...)

	if err := server.Connect(common.GetConnectionBackoff(emitterEventSource.ConnectionBackoff), func() error {
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
			logger.WithError(err).Errorln("failed to marshal the event data")
			return
		}

		logger.Infoln("dispatching event on data channel...")
		channels.Data <- eventBytes
	}); err != nil {
		return errors.Wrapf(err, "failed to subscribe to channel %s", emitterEventSource.ChannelName)
	}

	<-channels.Done

	logger.WithField("channel-name", emitterEventSource.ChannelName).Infoln("event source stopped, unsubscribe the channel")

	if err := client.Unsubscribe(emitterEventSource.ChannelKey, emitterEventSource.ChannelName); err != nil {
		logger.WithError(err).WithField("channel-name", emitterEventSource.ChannelName).Errorln("failed to unsubscribe")
	}

	return nil
}
