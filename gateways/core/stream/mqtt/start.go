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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// EventListener implements Eventing for mqtt event source
type EventListener struct {
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

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listens to events from a mqtt broker
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var mqttEventSource *v1alpha1.MQTTEventSource
	if err := yaml.Unmarshal(eventSource.Value, &mqttEventSource); err != nil {
		errorCh <- err
		return
	}

	logger = logger.WithFields(
		map[string]interface{}{
			common.LabelURL:      mqttEventSource.URL,
			common.LabelClientID: mqttEventSource.ClientId,
		},
	)

	logger.Infoln("setting up the message handler...")
	handler := func(c mqttlib.Client, msg mqttlib.Message) {
		logger.Infoln("dispatching event on data channel...")
		dataCh <- msg.Payload()
	}

	logger.Infoln("setting up the mqtt broker client...")
	opts := mqttlib.NewClientOptions().AddBroker(mqttEventSource.URL).SetClientID(mqttEventSource.ClientId)

	var client mqttlib.Client

	logger.Infoln("connecting to mqtt broker...")
	if err := gateways.Connect(&wait.Backoff{
		Factor:   mqttEventSource.ConnectionBackoff.Factor,
		Duration: mqttEventSource.ConnectionBackoff.Duration,
		Jitter:   mqttEventSource.ConnectionBackoff.Jitter,
		Steps:    mqttEventSource.ConnectionBackoff.Steps,
	}, func() error {
		client = mqttlib.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		return nil
	}); err != nil {
		logger.Info("failed to connect")
		errorCh <- err
		return
	}

	logger.Info("subscribing to the topic...")
	if token := client.Subscribe(mqttEventSource.Topic, 0, handler); token.Wait() && token.Error() != nil {
		logger.WithError(token.Error()).Error("failed to subscribe")
		errorCh <- token.Error()
		return
	}

	<-doneCh
	token := client.Unsubscribe(mqttEventSource.Topic)
	if token.Error() != nil {
		logger.WithError(token.Error()).Error("failed to unsubscribe client")
	}
}
