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

package redisstream

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for the Redis event source
type EventListener struct {
	EventSourceName string
	EventName       string
	EventSource     v1alpha1.RedisStreamEventSource
	Metrics         *metrics.Metrics
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
	return apicommon.RedisStreamEvent
}

// StartListening listens for new data on specified redis streams
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	log.Info("started processing the Redis stream event source...")
	defer sources.Recover(el.GetEventName())

	redisEventSource := &el.EventSource

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

	log.Infof("setting up a redis client for %s...", redisEventSource.HostAddress)
	client := redis.NewClient(opt)

	if status := client.Ping(); status.Err() != nil {
		return errors.Wrapf(status.Err(), "failed to connect to host %s and db %d for event source %s", redisEventSource.HostAddress, redisEventSource.DB, el.GetEventName())
	}
	log.Infof("connected to redis server %s", redisEventSource.HostAddress)

	// Create a common consumer group on all streams.
	// Only proceeds if all the streams are already present
	consumersGroup := "argo-events-cg"
	for _, stream := range redisEventSource.Streams {
		if err := client.XGroupCreate(stream, consumersGroup, "0").Err(); err != nil {
			// redis package doesn't seem to expose concrete error types
			if err.Error() != "BUSYGROUP Consumer Group name already exists" {
				return errors.Wrapf(err, "creating consumer group %s for stream %s on host %s for event source %s", consumersGroup, stream, redisEventSource.HostAddress, el.GetEventName())
			}
			log.Infof("Consumer group %s already exists", stream)
		}
	}

	readGroupArgs := make([]string, len(redisEventSource.Streams), 2*len(redisEventSource.Streams))
	copy(readGroupArgs, redisEventSource.Streams)
	for i := 0; i < len(redisEventSource.Streams); i++ {
		readGroupArgs = append(readGroupArgs, ">")
	}

	for {
		select {
		case <-ctx.Done():
			log.Info("Redis stream event source for host %s is stopped", redisEventSource.HostAddress)
			return nil
		default:
		}
		entries, err := client.XReadGroup(&redis.XReadGroupArgs{
			Group:    consumersGroup,
			Consumer: "argo-events-worker",
			Streams:  readGroupArgs,
			Count:    5,
			Block:    2 * time.Second,
			NoAck:    false,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return errors.Wrapf(err, "reading streams %s using XREADGROUP", strings.Join(redisEventSource.Streams, " "))
		}

		for _, entry := range entries {
			for _, message := range entry.Messages {
				if err := el.handleOne(entry.Stream, message, dispatch, log); err != nil {
					log.With("stream", entry.Stream, "message_id", message.ID).Errorw("failed to process Redis stream message", zap.Error(err))
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
					continue
				}
				if err := client.XAck(entry.Stream, consumersGroup, message.ID).Err(); err != nil {
					log.With("stream", entry.Stream, "message_id", message.ID).Errorw("failed to acknowledge Redis stream message", zap.Error(err))
				}
			}
		}

	}
}

func (el *EventListener) handleOne(stream string, message redis.XMessage, dispatch func([]byte) error, log *zap.SugaredLogger) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	log.With("stream", stream, "message_id", message.ID).Info("received a message")
	eventData := &events.RedisStreamEventData{
		Stream:   stream,
		Id:       message.ID,
		Values:   message.Values,
		Metadata: el.EventSource.Metadata,
	}
	eventBody, err := json.Marshal(&eventData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the event data, rejecting the event...")
	}
	log.With("stream", stream).Info("dispatching the event on the data channel...")
	if err = dispatch(eventBody); err != nil {
		return errors.Wrap(err, "failed dispatch a Redis stream event")
	}
	return nil
}
