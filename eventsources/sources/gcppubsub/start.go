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

	"cloud.google.com/go/compute/metadata"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	err := el.fillDefault(logger)
	if err != nil {
		return errors.Wrapf(err, "failed to fill default values for %s", el.GetEventName())
	}

	pubsubEventSource := &el.PubSubEventSource
	log := logger.With(
		"topic", pubsubEventSource.Topic,
		"topicProjectID", pubsubEventSource.TopicProjectID,
		"projectID", pubsubEventSource.ProjectID,
		"subscriptionID", pubsubEventSource.SubscriptionID,
	).Desugar()

	if pubsubEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Info("setting up a client to connect to PubSub...")
	client, subscription, err := el.prepareSubscription(ctx, log)
	if err != nil {
		return errors.Wrapf(err, "failed to prepare client or subscription for %s", el.GetEventName())
	}

	log.Info("listening for messages from PubSub...")
	err = subscription.Receive(ctx, func(msgCtx context.Context, m *pubsub.Message) {
		log.Info("received GCP PubSub Message from topic")
		eventData := &events.PubSubEventData{
			ID:          m.ID,
			Body:        m.Data,
			Attributes:  m.Attributes,
			PublishTime: m.PublishTime.String(),
			Metadata:    pubsubEventSource.Metadata,
		}
		if pubsubEventSource.JSONBody {
			eventData.Body = (*json.RawMessage)(&m.Data)
		}
		eventBytes, err := json.Marshal(eventData)
		if err != nil {
			log.Error("failed to marshal the event data", zap.Error(err))
			m.Nack()
			return
		}

		log.Info("dispatching event...")
		err = dispatch(eventBytes)
		if err != nil {
			log.Error("failed to dispatch GCP PubSub event", zap.Error(err))
			m.Nack()
			return
		}
		m.Ack()
	})
	if err != nil {
		return errors.Wrapf(err, "failed to receive the messages for subscription %s for %s", subscription, el.GetEventName())
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

func (el *EventListener) fillDefault(logger *zap.SugaredLogger) error {
	// Default value for each field
	//  - ProjectID:        determine from GCP metadata server (only valid in GCP)
	//  - TopicProjectID:   same as ProjectID (filled only if topic is specified)
	//  - SubscriptionID:   name + hash suffix
	//  - Topic:            nothing (fine if subsc. exists, otherwise fail)

	if el.PubSubEventSource.ProjectID == "" {
		logger.Debug("determine project ID from GCP metadata server")
		proj, err := metadata.ProjectID()
		if err != nil {
			return errors.Wrap(err, "project ID is not given and couldn't determine from GCP metadata server")
		}
		el.PubSubEventSource.ProjectID = proj
	}

	if el.PubSubEventSource.TopicProjectID == "" && el.PubSubEventSource.Topic != "" {
		el.PubSubEventSource.TopicProjectID = el.PubSubEventSource.ProjectID
	}

	if el.PubSubEventSource.SubscriptionID == "" {
		logger.Debug("auto generate subscription ID")
		hashcode, err := el.hash()
		if err != nil {
			return errors.Wrap(err, "failed get hashcode")
		}
		el.PubSubEventSource.SubscriptionID = fmt.Sprintf("%s-%s", el.GetEventName(), hashcode)
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

func (el *EventListener) prepareSubscription(ctx context.Context, logger *zap.Logger) (*pubsub.Client, *pubsub.Subscription, error) {
	pubsubEventSource := &el.PubSubEventSource

	opts := make([]option.ClientOption, 0, 1)
	if secret := el.PubSubEventSource.CredentialSecret; secret != nil {
		logger.Debug("using credentials from secret")
		jsonCred, err := common.GetSecretFromVolume(secret)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not find credentials")
		}
		opts = append(opts, option.WithCredentialsJSON([]byte(jsonCred)))
	} else if credFile := el.PubSubEventSource.DeprecatedCredentialsFile; credFile != "" {
		logger.Debug("using credentials from file (DEPRECATED)")
		opts = append(opts, option.WithCredentialsFile(credFile))
	} else {
		logger.Debug("using default credentials")
	}
	client, err := pubsub.NewClient(ctx, pubsubEventSource.ProjectID, opts...)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to set up client for %s", el.GetEventName())
	}
	logger.Debug("set up pubsub client")

	subscription := client.Subscription(pubsubEventSource.SubscriptionID)

	// Overall logics are as follows:
	//
	// subsc. exists | topic given | topic exists | action                | required permissions
	// :------------ | :---------- | :----------- | :-------------------- | :-----------------------------------------------------------------------------
	// no            | no          | -            | invalid               | -
	// yes           | no          | -            | do nothing            | nothing extra
	// yes           | yes         | -            | verify topic          | pubsub.subscriptions.get (subsc.)
	// no            | yes         | yes          | create subsc.         | pubsub.subscriptions.create (proj.) + pubsub.topics.attachSubscription (topic)
	// no            | yes         | no           | create topic & subsc. | above + pubsub.topics.create (proj. for topic)

	// trick: you don't need to have get permission to check only whether it exists
	perms, err := subscription.IAM().TestPermissions(ctx, []string{"pubsub.subscriptions.consume"})
	subscExists := len(perms) == 1
	if !subscExists {
		switch status.Code(err) {
		case codes.OK:
			client.Close()
			return nil, nil, errors.Errorf("you lack permission to pull from %s", subscription)
		case codes.NotFound:
			// OK, maybe the subscription doesn't exist yet, so create it later
			// (it possibly means project itself doesn't exist, but it's ok because we'll see an error later in such case)
		default:
			client.Close()
			return nil, nil, errors.Wrapf(err, "failed to test permission for subscription %s", subscription)
		}
	}
	logger.Debug("checked if subscription exists and you have right permission")

	// subsc. exists | topic given | topic exists | action                | required permissions
	// :------------ | :---------- | :----------- | :-------------------- | :-----------------------------------------------------------------------------
	// no            | no          | -            | invalid               | -
	// yes           | no          | -            | do nothing            | nothing extra
	if pubsubEventSource.Topic == "" {
		if !subscExists {
			client.Close()
			return nil, nil, errors.Errorf("you need to specify topicID to create missing subscription %s", subscription)
		}
		logger.Debug("subscription exists and no topic given, fine")
		return client, subscription, nil
	}

	// subsc. exists | topic given | topic exists | action                | required permissions
	// :------------ | :---------- | :----------- | :-------------------- | :-----------------------------------------------------------------------------
	// yes           | yes         | -            | verify topic          | pubsub.subscriptions.get (subsc.)
	topic := client.TopicInProject(pubsubEventSource.Topic, pubsubEventSource.TopicProjectID)

	if subscExists {
		subscConfig, err := subscription.Config(ctx)
		if err != nil {
			client.Close()
			return nil, nil, errors.Wrapf(err, "failed to get subscription's config for verifying topic")
		}
		switch actualTopic := subscConfig.Topic.String(); actualTopic {
		case "_deleted-topic_":
			client.Close()
			return nil, nil, errors.New("the topic for the subscription has been deleted")
		case topic.String():
			logger.Debug("subscription exists and its topic matches given one, fine")
			return client, subscription, nil
		default:
			client.Close()
			return nil, nil, errors.Errorf("this subscription belongs to wrong topic %s", actualTopic)
		}
	}

	// subsc. exists | topic given | topic exists | action                | required permissions
	// :------------ | :---------- | :----------- | :-------------------- | :-----------------------------------------------------------------------------
	// no            | yes         | ???          | create subsc.         | pubsub.subscriptions.create (proj.) + pubsub.topics.attachSubscription (topic)
	//                               ↑ We don't know yet, but just try to create subsc.
	logger.Debug("subscription doesn't seem to exist")
	_, err = client.CreateSubscription(ctx, subscription.ID(), pubsub.SubscriptionConfig{Topic: topic})
	switch status.Code(err) {
	case codes.OK:
		logger.Debug("subscription created")
		return client, subscription, nil
	case codes.NotFound:
		// OK, maybe the topic doesn't exist yet, so create it later
		// (it possibly means project itself doesn't exist, but it's ok because we'll see an error later in such case)
	default:
		client.Close()
		return nil, nil, errors.Wrapf(err, "failed to create %s for %s", subscription, topic)
	}

	// subsc. exists | topic given | topic exists | action                | required permissions
	// :------------ | :---------- | :----------- | :-------------------- | :-----------------------------------------------------------------------------
	// no            | yes         | no           | create topic & subsc. | above + pubsub.topics.create (proj. for topic)
	logger.Debug("topic doesn't seem to exist neither")
	// NB: you need another client for topic because it might be in different project
	topicClient, err := pubsub.NewClient(ctx, pubsubEventSource.TopicProjectID, opts...)
	if err != nil {
		client.Close()
		return nil, nil, errors.Wrapf(err, "failed to create client to create %s", topic)
	}
	defer topicClient.Close()

	_, err = topicClient.CreateTopic(ctx, topic.ID())
	if err != nil {
		client.Close()
		return nil, nil, errors.Wrapf(err, "failed to create %s", topic)
	}
	logger.Debug("topic created")
	_, err = client.CreateSubscription(ctx, subscription.ID(), pubsub.SubscriptionConfig{Topic: topic})
	if err != nil {
		client.Close()
		return nil, nil, errors.Wrapf(err, "failed to create %s for %s", subscription, topic)
	}
	logger.Debug("subscription created")
	return client, subscription, nil
}
