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
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// EventSourceListener implements Eventing
type EventSourceListener struct {
	Logger *logrus.Logger
}

// StartEventSource starts processing the GCP PubSub event source
func (listener *EventSourceListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	listener.Logger.WithField(common.LabelEventSource, eventSource.Name).Infoln("started processing the event source...")

	ctx := eventStream.Context()

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(ctx, eventSource, dataCh, errorCh, doneCh)
	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

// listenEvents listens to GCP PubSub events
func (listener *EventSourceListener) listenEvents(ctx context.Context, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	var pubsubEventSource *v1alpha1.PubSubEventSource
	if err := yaml.Unmarshal(eventSource.Value, &pubsubEventSource); err != nil {
		listener.Logger.WithError(err).WithField(common.LabelEventSource, eventSource.Name).Errorln("failed to parse the event source")
		errorCh <- err
		return
	}

	// Create a new topic with the given name if none exists
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name).WithField("topic", pubsubEventSource.Topic)

	client, err := pubsub.NewClient(ctx, pubsubEventSource.ProjectID, option.WithCredentialsFile(pubsubEventSource.CredentialsFile))
	if err != nil {
		errorCh <- err
		return
	}

	topicClient := client // use same client for topic and subscription by default
	if pubsubEventSource.TopicProjectID != "" && pubsubEventSource.TopicProjectID != pubsubEventSource.ProjectID {
		topicClient, err = pubsub.NewClient(ctx, pubsubEventSource.TopicProjectID, option.WithCredentialsFile(pubsubEventSource.CredentialsFile))
		if err != nil {
			errorCh <- err
			return
		}
	}

	topic := topicClient.Topic(pubsubEventSource.Topic)
	exists, err := topic.Exists(ctx)
	if err != nil {
		errorCh <- err
		return
	}
	if !exists {
		logger.Info("Creating GCP PubSub topic")
		if _, err := topicClient.CreateTopic(ctx, pubsubEventSource.Topic); err != nil {
			errorCh <- err
			return
		}
	}

	logger.Info("Subscribing to GCP PubSub topic")
	subscriptionName := fmt.Sprintf("%s-%s", eventSource.Name, eventSource.Id)
	subscription := client.Subscription(subscriptionName)
	exists, err = subscription.Exists(ctx)

	if err != nil {
		errorCh <- err
		return
	}
	if exists {
		logger.Warn("Using an existing subscription")
	} else {
		logger.Info("Creating subscription")
		if _, err := client.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{Topic: topic}); err != nil {
			errorCh <- err
			return
		}
	}

	err = subscription.Receive(ctx, func(msgCtx context.Context, m *pubsub.Message) {
		logger.Info("received GCP PubSub Message from topic")
		dataCh <- m.Data
		m.Ack()
	})
	if err != nil {
		errorCh <- err
		return
	}

	<-doneCh

	// after this point, panic on errors
	logger.Info("deleting GCP PubSub subscription")
	if err = subscription.Delete(context.Background()); err != nil {
		panic(err)
	}

	logger.Info("closing GCP PubSub client")
	if err = client.Close(); err != nil {
		panic(err)
	}
}
