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

package gcppubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// EventListener implements Eventing for gcp pub-sub event source
type EventListener struct {
	EventSourceName   string
	EventName         string
	PubSubEventSource v1alpha1.PubSubEventSource
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
	return apicommon.PubSubEvent
}

// StartListening listens to GCP PubSub events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	// In order to listen events from GCP PubSub,
	// 1. Parse the event source that contains configuration to connect to GCP PubSub
	// 2. Create a new PubSub client
	// 3. Create the topic if one doesn't exist already
	// 4. Create a subscription if one doesn't exist already.
	// 5. Start listening to messages on the queue
	// 6. Once the event source is stopped perform cleaning up - 1. Delete the subscription if configured so 2. Close the PubSub client

	logger := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	logger.Info("started processing the GCP Pub Sub event source...")
	defer sources.Recover(el.GetEventName())

	pubsubEventSource := &el.PubSubEventSource

	if pubsubEventSource.JSONBody {
		logger.Info("assuming all events have a json body...")
	}

	logger = logger.With("topic", pubsubEventSource.Topic)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("setting up a client to connect to PubSub...")

	var opt []option.ClientOption
	projectID := pubsubEventSource.ProjectID

	if !pubsubEventSource.EnableWorkloadIdentity {
		opt = append(opt, option.WithCredentialsFile(pubsubEventSource.CredentialsFile))
	}

	// Use default ProjectID unless TopicProjectID exists
	if pubsubEventSource.TopicProjectID != "" && pubsubEventSource.TopicProjectID != pubsubEventSource.ProjectID {
		projectID = pubsubEventSource.TopicProjectID
	}

	// Create a new topic with the given name if none exists
	client, err := pubsub.NewClient(ctx, projectID, opt...)
	if err != nil {
		return errors.Wrapf(err, "failed to set up client for %s", el.GetEventName())
	}

	logger.Info("getting topic information from PubSub...")
	topic := client.Topic(pubsubEventSource.Topic)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get status of the topic %s for %s", pubsubEventSource.Topic, el.GetEventName())
	}
	if !exists {
		logger.Info("topic doesn't exist, creating the PubSub topic...")
		if _, err := client.CreateTopic(ctx, pubsubEventSource.Topic); err != nil {
			return errors.Wrapf(err, "failed to create the topic %s for %s", pubsubEventSource.Topic, el.GetEventName())
		}
	}

	hashcode, err := el.hash()
	if err != nil {
		logger.Desugar().Error("failed get hashcode", zap.Error(err))
		return err
	}
	subscriptionName := fmt.Sprintf("%s-%s", el.GetEventName(), hashcode)

	log := logger.With("subscription", subscriptionName).Desugar()

	log.Info("subscribing to PubSub topic...")
	subscription := client.Subscription(subscriptionName)
	exists, err = subscription.Exists(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get status of the subscription %s for %s", subscriptionName, el.GetEventName())
	}
	if exists {
		log.Warn("using an existing subscription...")
	} else {
		log.Info("creating a new subscription...")
		if _, err := client.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{Topic: topic}); err != nil {
			return errors.Wrapf(err, "failed to create the subscription %s for %s", subscriptionName, el.GetEventName())
		}
	}

	log.Info("listening for messages from PubSub...")
	err = subscription.Receive(ctx, func(msgCtx context.Context, m *pubsub.Message) {
		log.Info("received GCP PubSub Message from topic")
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
			log.Error("failed to marshal the event data", zap.Error(err))
			return
		}

		log.Info("dispatching event...")
		err = dispatch(eventBytes)
		if err != nil {
			log.Error("failed to dispatch GCP PubSub event", zap.Error(err))
			return
		}
		m.Ack()
	})
	if err != nil {
		return errors.Wrapf(err, "failed to receive the messages for subscription %s and topic %s for %s", subscriptionName, pubsubEventSource.Topic, el.GetEventName())
	}

	<-ctx.Done()

	log.Info("event source has been stopped")

	if pubsubEventSource.DeleteSubscriptionOnFinish {
		log.Info("deleting PubSub subscription...")
		if err = subscription.Delete(context.Background()); err != nil {
			log.Error("failed to delete the PubSub subscription", zap.Error(err))
		}
	}

	log.Info("closing PubSub client...")
	if err = client.Close(); err != nil {
		log.Error("failed to close the PubSub client", zap.Error(err))
	}

	return nil
}

func (el *EventListener) hash() (string, error) {
	body, err := json.Marshal(&el.PubSubEventSource)
	if err != nil {
		return "", err
	}
	return common.Hasher(el.GetEventName() + string(body)), nil
}
