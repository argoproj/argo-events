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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
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
	Logger *logrus.Logger
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

// listenEvents listens events from emitter subscription
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	logger := listener.Logger.WithField(common.LabelEventSourceName, eventSource.Name)

	logger.Infoln("parsing the event source")

	var emitterEventSource *v1alpha1.EmitterEventSource
	if err := yaml.Unmarshal(eventSource.Value, &emitterEventSource); err != nil {
		errorCh <- errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
		return
	}

	var options []func(client *emitter.Client)
	options = append(options, emitter.WithBrokers(emitterEventSource.Broker), emitter.WithAutoReconnect(true))

	logger.WithField("secret-name", emitterEventSource.ChannelKey.Name).Infoln("retrieving the channel key")
	channelKey, err := common.GetSecrets(listener.K8sClient, emitterEventSource.Namespace, emitterEventSource.ChannelKey)
	if err != nil {
		errorCh <- errors.Wrapf(err, "failed to retrieve the channel key from %s", emitterEventSource.ChannelKey.Name)
		return
	}

	if emitterEventSource.Username != nil {
		username, err := common.GetSecrets(listener.K8sClient, emitterEventSource.Namespace, emitterEventSource.Username)
		if err != nil {
			errorCh <- errors.Wrapf(err, "failed to retrieve the username from %s", emitterEventSource.Username.Name)
			return
		}
		options = append(options, emitter.WithUsername(username))
	}

	if emitterEventSource.Password != nil {
		password, err := common.GetSecrets(listener.K8sClient, emitterEventSource.Namespace, emitterEventSource.Password)
		if err != nil {
			errorCh <- errors.Wrapf(err, "failed to retrieve the password from %s", emitterEventSource.Password.Name)
			return
		}
		options = append(options, emitter.WithPassword(password))
	}

	logger.WithField("channel-name", emitterEventSource.ChannelName).Infoln("creating a client")
	client := emitter.NewClient(options...)

	if err := client.Connect(); err != nil {
		errorCh <- errors.Wrapf(err, "failed to create client for channel %s", emitterEventSource.ChannelName)
		return
	}

	if err := client.Subscribe(channelKey, emitterEventSource.ChannelName, func(_ *emitter.Client, message emitter.Message) {
		logger.Infoln("dispatching event on data channel...")
		dataCh <- message.Payload()
	}); err != nil {
		errorCh <- errors.Wrapf(err, "failed to subscribe to channel %s", emitterEventSource.ChannelName)
		return
	}

	<-doneCh
	logger.WithField("channel-name", emitterEventSource.ChannelName).Infoln("event source stopped, unsubscribe the channel")
	if err := client.Unsubscribe(channelKey, emitterEventSource.ChannelName); err != nil {
		logger.WithError(err).WithField("channel-name", emitterEventSource.ChannelName).Errorln("failed to unsubscribe")
	}
}
