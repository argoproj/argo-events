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
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
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

// listenEvents listens to GCP PubSub events
func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	// In order to listen events from GCP PubSub,
	// 1. Parse the event source that contains configuration to connect to GCP PubSub
	// 2. Create a new PubSub client
	// 3. Create the topic if one doesn't exist already
	// 4. Create a subscription if one doesn't exist already.
	// 5. Start listening to messages on the queue
	// 6. Once the event source is stopped perform cleaning up - 1. Delete the subscription if configured so 2. Close the PubSub client

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the PubSub event source...")
	var pubsubEventSource *v1alpha1.PubSubEventSource
	if err := yaml.Unmarshal(eventSource.Value, &pubsubEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse the event source %s", eventSource.Name)
	}

	if pubsubEventSource.JSONBody {
		logger.Infoln("assuming all events have a json body...")
	}

	logger = logger.WithField("topic", pubsubEventSource.Topic)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Infoln("setting up a client to connect to PubSub...")

	var opt []option.ClientOption
	projectId := pubsubEventSource.ProjectID

	if !pubsubEventSource.EnableWorkflowIdentity {
		opt = append(opt, option.WithCredentialsFile(pubsubEventSource.CredentialsFile))
	}

	// Use default ProjectID unless TopicProjectID exists
	if pubsubEventSource.TopicProjectID != "" && pubsubEventSource.TopicProjectID != pubsubEventSource.ProjectID {
		projectId = pubsubEventSource.TopicProjectID
	}

	// Create a new topic with the given name if none exists
	client, err := pubsub.NewClient(ctx, projectId, opt...)
	if err != nil {
		return errors.Wrapf(err, "failed to set up client for %s", eventSource.Name)
	}

	logger.Infoln("getting topic information from PubSub...")
	topic := client.Topic(pubsubEventSource.Topic)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get status of the topic %s for %s", pubsubEventSource.Topic, eventSource.Name)
	}
	if !exists {
		logger.Infoln("topic doesn't exist, creating the PubSub topic...")
		if _, err := client.CreateTopic(ctx, pubsubEventSource.Topic); err != nil {
			return errors.Wrapf(err, "failed to create the topic %s for %s", pubsubEventSource.Topic, eventSource.Name)
		}
	}

	subscriptionName := fmt.Sprintf("%s-%s", eventSource.Name, eventSource.Id)

	logger = logger.WithField("subscription", subscriptionName)

	logger.Infoln("subscribing to PubSub topic...")
	subscription := client.Subscription(subscriptionName)
	exists, err = subscription.Exists(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get status of the subscription %s for %s", subscriptionName, eventSource.Name)
	}
	if exists {
		logger.Warnln("using an existing subscription...")
	} else {
		logger.Infoln("creating a new subscription...")
		if _, err := client.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{Topic: topic}); err != nil {
			return errors.Wrapf(err, "failed to create the subscription %s for %s", subscriptionName, eventSource.Name)
		}
	}

	logger.Infoln("listening for messages from PubSub...")
	err = subscription.Receive(ctx, func(msgCtx context.Context, m *pubsub.Message) {
		logger.Info("received GCP PubSub Message from topic")
		eventData := &events.PubSubEventData{
			ID:          m.ID,
			Body:        m.Data,
			Attributes:  m.Attributes,
			PublishTime: m.PublishTime.String(),
		}
		if pubsubEventSource.JSONBody {
			eventData.Body = (*json.RawMessage)(&m.Data)
		}
		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			logger.WithError(err).Errorln("failed to marshal the event data")
			return
		}

		logger.Info("sending event on the data channel...")
		channels.Data <- eventBytes
		m.Ack()
	})
	if err != nil {
		return errors.Wrapf(err, "failed to receive the messages for subscription %s and topic %s for %s", subscriptionName, pubsubEventSource.Topic, eventSource.Name)
	}

	<-channels.Done

	logger.Infoln("event source has been stopped")

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

	return nil
}
