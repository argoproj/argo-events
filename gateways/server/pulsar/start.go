/*
Copyright 2019 Lucidworks Inc.

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

package pulsar

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"time"
)

// EventListener implements Eventing for Pulsar event source
type EventListener struct {
	Logger *logrus.Logger
	// k8sClient is kubernetes client
	K8sClient kubernetes.Interface
}

// StartEventSource starts an event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer server.Recover(eventSource.Name)

	logrus.WithField(common.LabelEventSource, eventSource.Name).Info(">> StartEventSource")

	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("started processing the event source...")

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)
	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents fires an event when interval completes and item is processed from queue.
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {

	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)
	log.Info(">> listenEvents...") // todo debug

	var pulsarEventSource *v1alpha1.PulsarEventSource
	if err := yaml.Unmarshal(eventSource.Value, &pulsarEventSource); err != nil {
		errorCh <- err
		return
	}

	pulsarClient, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               pulsarEventSource.PulsarConfigUrl,
		ConnectionTimeout: parseDuration("ConnectionTimeout", pulsarEventSource.PulsarConfigConnectionTimeout, "30s"),
		OperationTimeout:  parseDuration("OperationTimeout", pulsarEventSource.PulsarConfigOperationTimeout, "30s")})
	if err != nil {
		log.WithError(err).Error("Cannot connect to Pulsar")
		errorCh <- err
		return
	} else {
		log.WithField("url", pulsarEventSource.PulsarConfigUrl).Info("Created Pulsar client")
	}
	defer pulsarClient.Close()

	topicLog := log.WithFields(logrus.Fields{
		"topic":            pulsarEventSource.PulsarConfigTopic,
		"subscriptionName": pulsarEventSource.PulsarConfigSubscriptionName,
		"subscriptionType": pulsarEventSource.PulsarConfigSubscriptionType})

	consumer, err := pulsarClient.Subscribe(pulsar.ConsumerOptions{
		Topic:            pulsarEventSource.PulsarConfigTopic,
		SubscriptionName: pulsarEventSource.PulsarConfigSubscriptionName,
		Type:             parseSubscriptionType(pulsarEventSource.PulsarConfigSubscriptionType),
	})
	if err != nil {
		topicLog.WithError(err).Error("Cannot subscribe to Pulsar topic")
		errorCh <- err
		return
	} else {
		topicLog.Info("Subscribed to Pulsar topic")
	}
	defer consumer.Close()

	messages := consumer.Chan()

	log.Info("Starting listening to Pulsar events...")

	for {
		select {
		case msg := <-messages:
			log.WithFields(logrus.Fields{"topic": msg.Topic(), "id": msg.ID()}).Info("Message received") // todo debug
			dataCh <- msg.Payload()
		case <-doneCh:
			log.Info("Done event is received, closing Pulsar client...")
			consumer.Close()
			pulsarClient.Close()
			return
		}
	}
}

func parseDuration(name string, s string, defaultValue string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err == nil {
		return duration
	}
	logrus.WithField(name, s).Warn("Cannot parse the duration - using the default value " + defaultValue)
	duration, err = time.ParseDuration(defaultValue)
	if err == nil {
		return duration
	} else {
		// that is unlikely since the default value _should_ be parsable, but since it is not, defaulting to 30 seconds
		duration, _ = time.ParseDuration("30s")
		return duration
	}
}

func parseSubscriptionType(s string) pulsar.SubscriptionType {
	switch s {
	case "Shared":
		return pulsar.Shared
	case "Exclusive":
		return pulsar.Exclusive
	case "Failover":
		return pulsar.Failover
	case "KeyShared":
		return pulsar.KeyShared
	default:
		return pulsar.Exclusive
	}
}
