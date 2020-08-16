/*
Copyright 2020 BlackRock, Inc.

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

package redis

import (
	"context"
	"encoding/json"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// EventListener implements Eventing for the Redis event source
type EventListener struct {
	EventSourceName  string
	EventName        string
	RedisEventSource v1alpha1.RedisEventSource
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
	return apicommon.RedisEvent
}

// StartListening listens events published by redis
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the Redis event source...")
	defer sources.Recover(el.GetEventName())

	redisEventSource := &el.RedisEventSource

	opt := &redis.Options{
		Addr: redisEventSource.HostAddress,
		DB:   int(redisEventSource.DB),
	}

	log.Info("retrieving password if it has been configured...")
	if redisEventSource.Password != nil {
		password, err := common.GetSecretFromVolume(redisEventSource.Password)
		if err != nil {
			return errors.Wrapf(err, "failed to find the secret password %s", redisEventSource.Password.Name)
		}
		opt.Password = password
	}

	if redisEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(redisEventSource.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		opt.TLSConfig = tlsConfig
	}

	log.Info("setting up a redis client...")
	client := redis.NewClient(opt)

	if status := client.Ping(); status.Err() != nil {
		return errors.Wrapf(status.Err(), "failed to connect to host %s and db %d for event source %s", redisEventSource.HostAddress, redisEventSource.DB, el.GetEventName())
	}

	pubsub := client.Subscribe(redisEventSource.Channels...)
	// Wait for confirmation that subscription is created before publishing anything.
	if _, err := pubsub.Receive(); err != nil {
		return errors.Wrapf(err, "failed to receive the subscription confirmation for event source %s", el.GetEventName())
	}

	// Go channel which receives messages.
	ch := pubsub.Channel()
	for {
		select {
		case message := <-ch:
			log.With("channel", message.Channel).Info("received a message")
			eventData := &events.RedisEventData{
				Channel:  message.Channel,
				Pattern:  message.Pattern,
				Body:     message.Payload,
				Metadata: redisEventSource.Metadata,
			}
			eventBody, err := json.Marshal(&eventData)
			if err != nil {
				log.With("channel", message.Channel).Desugar().Error("failed to marshal the event data, rejecting the event...", zap.Error(err))
				continue
			}
			log.With("channel", message.Channel).Info("dispatching th event on the data channel...")
			err = dispatch(eventBody)
			if err != nil {
				log.With("channel", message.Channel).Desugar().Error("failed to dispatch redis event", zap.Error(err))
			}
		case <-ctx.Done():
			log.Info("event source is stopped. unsubscribing the subscription")
			if err := pubsub.Unsubscribe(redisEventSource.Channels...); err != nil {
				log.Desugar().Error("failed to unsubscribe", zap.Error(err))
			}
			return nil
		}
	}
}
