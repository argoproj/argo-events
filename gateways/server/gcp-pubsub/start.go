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

package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// EventListener implements Eventing for gcp pub-sub event source
type EventListener struct {
	// Logger to log stuff
	Logger *logrus.Logger
}

// StartEventSource starts processing the GCP PubSub event source
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer server.Recover(eventSource.Name)

	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	ctx := eventStream.Context()

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(ctx, eventSource, dataCh, errorCh, doneCh)
	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listens to GCP PubSub events
func (listener *EventListener) listenEvents(ctx context.Context, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	// In order to listen events from GCP PubSub,
	// 1. Parse the event source that contains configuration to connect to GCP PubSub
	// 2. Create a new PubSub client
	// 3. Create the topic if one doesn't exist already
	// 4. Create a subscription if one doesn't exist already.
	// 5. Start listening to messages on the queue
	// 6. Once the event source is stopped perform cleaning up - 1. Delete the subscription if configured so 2. Close the PubSub client

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing PubSub event source...")
	var pubsubEventSource *v1alpha1.PubSubEventSource
	if err := yaml.Unmarshal(eventSource.Value, &pubsubEventSource); err != nil {
		errorCh <- err
		return
	}

	logger = logger.WithField("topic", pubsubEventSource.Topic)

	logger.Infoln("setting up a client to connect to PubSub...")

	// Create a new topic with the given name if none exists
	client, err := pubsub.NewClient(ctx, pubsubEventSource.ProjectID, option.WithCredentialsFile(pubsubEventSource.CredentialsFile))
	if err != nil {
		errorCh <- err
		return
	}

	// use same client for topic and subscription by default
	topicClient := client
	if pubsubEventSource.TopicProjectID != "" && pubsubEventSource.TopicProjectID != pubsubEventSource.ProjectID {
		topicClient, err = pubsub.NewClient(ctx, pubsubEventSource.TopicProjectID, option.WithCredentialsFile(pubsubEventSource.CredentialsFile))
		if err != nil {
			errorCh <- err
			return
		}
	}

	logger.Infoln("getting topic information from PubSub...")
	topic := topicClient.Topic(pubsubEventSource.Topic)
	exists, err := topic.Exists(ctx)
	if err != nil {
		errorCh <- err
		return
	}
	if !exists {
		logger.Infoln("topic doesn't exist, creating the GCP PubSub topic...")
		if _, err := topicClient.CreateTopic(ctx, pubsubEventSource.Topic); err != nil {
			errorCh <- err
			return
		}
	}

	subscriptionName := fmt.Sprintf("%s-%s", eventSource.Name, eventSource.Id)

	logger = logger.WithField("subscription", subscriptionName)

	logger.Infoln("subscribing to PubSub topic...")
	subscription := client.Subscription(subscriptionName)
	exists, err = subscription.Exists(ctx)

	if err != nil {
		errorCh <- err
		return
	}
	if exists {
		logger.Warnln("using an existing subscription...")
	} else {
		logger.Infoln("creating a new subscription...")
		if _, err := client.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{Topic: topic}); err != nil {
			errorCh <- err
			return
		}
	}

	logger.Infoln("listening for messages from PubSub...")
	err = subscription.Receive(ctx, func(msgCtx context.Context, m *pubsub.Message) {
		logger.Info("received GCP PubSub Message from topic")
		eventData := &Data{
			ID:          m.ID,
			Data:        m.Data,
			Attributes:  m.Attributes,
			PublishTime: m.PublishTime.String(),
		}
		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			logger.WithError(err).Errorln("failed to marshal the event data")
			return
		}
		dataCh <- eventBytes
		m.Ack()
	})
	if err != nil {
		errorCh <- err
		return
	}

	<-doneCh

	if pubsubEventSource.DeleteSubscriptionOnFinish {
		logger.Info("deleting PubSub subscription...")
		if err = subscription.Delete(context.Background()); err != nil {
			logger.WithError(err).Errorln("failed to delete the PubSub subscription")
		}
	}

	logger.Info("closing PubSub client...")
	if err = client.Close(); err != nil {
		logger.WithError(err).Errorln("failed to close the PubSub client")
	}
}
