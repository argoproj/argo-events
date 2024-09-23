/*
Copyright 2020 The Argoproj Authors.

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
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
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
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.RedisStreamEvent
}

// StartListening listens for new data on specified redis streams
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
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
		password, err := sharedutil.GetSecretFromVolume(redisEventSource.Password)
		if err != nil {
			return fmt.Errorf("failed to find the secret password %s, %w", redisEventSource.Password.Name, err)
		}
		opt.Password = password
	}

	if redisEventSource.Username != "" {
		opt.Username = redisEventSource.Username
	}

	if redisEventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(redisEventSource.TLS)
		if err != nil {
			return fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		opt.TLSConfig = tlsConfig
	}

	log.Infof("setting up a redis client for %s...", redisEventSource.HostAddress)
	client := redis.NewClient(opt)

	if status := client.Ping(ctx); status.Err() != nil {
		return fmt.Errorf("failed to connect to host %s and db %d for event source %s, %w", redisEventSource.HostAddress, redisEventSource.DB, el.GetEventName(), status.Err())
	}
	log.Infof("connected to redis server %s", redisEventSource.HostAddress)

	// Create a common consumer group on all streams to start reading from beginning of the streams.
	// Only proceeds if all the streams are already present
	consumersGroup := "argo-events-cg"
	if len(redisEventSource.ConsumerGroup) != 0 {
		consumersGroup = redisEventSource.ConsumerGroup
	}
	for _, stream := range redisEventSource.Streams {
		// create a consumer group to start reading from the current last entry in the stream (https://redis.io/commands/xgroup-create)
		if err := client.XGroupCreate(ctx, stream, consumersGroup, "$").Err(); err != nil {
			// redis package doesn't seem to expose concrete error types
			if err.Error() != "BUSYGROUP Consumer Group name already exists" {
				return fmt.Errorf("creating consumer group %s for stream %s on host %s for event source %s, %w", consumersGroup, stream, redisEventSource.HostAddress, el.GetEventName(), err)
			}
			log.Infof("Consumer group %q already exists in stream %q", consumersGroup, stream)
		}
	}

	readGroupArgs := make([]string, 2*len(redisEventSource.Streams))
	copy(readGroupArgs, redisEventSource.Streams)
	// Start by reading our pending messages(previously read but not acknowledged).
	streamToLastEntryMapping := make(map[string]string, len(redisEventSource.Streams))
	for _, s := range redisEventSource.Streams {
		streamToLastEntryMapping[s] = "0-0"
	}

	updateReadGroupArgs := func() {
		for i, s := range redisEventSource.Streams {
			readGroupArgs[i+len(redisEventSource.Streams)] = streamToLastEntryMapping[s]
		}
	}
	updateReadGroupArgs()

	msgCount := redisEventSource.MaxMsgCountPerRead
	if msgCount == 0 {
		msgCount = 10
	}

	var msgsToAcknowledge []string
	for {
		select {
		case <-ctx.Done():
			log.Infof("Redis stream event source for host %s is stopped", redisEventSource.HostAddress)
			return nil
		default:
		}
		entries, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumersGroup,
			Consumer: "argo-events-worker",
			Streams:  readGroupArgs,
			Count:    int64(msgCount),
			Block:    2 * time.Second,
			NoAck:    false,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			log.With("streams", redisEventSource.Streams).Errorw("reading streams using XREADGROUP", zap.Error(err))
		}

		for _, entry := range entries {
			if len(entry.Messages) == 0 {
				// Completed consuming pending messages. Now start consuming new messages
				streamToLastEntryMapping[entry.Stream] = ">"
			}

			msgsToAcknowledge = msgsToAcknowledge[:0]

			for _, message := range entry.Messages {
				if err := el.handleOne(entry.Stream, message, dispatch, log); err != nil {
					log.With("stream", entry.Stream, "message_id", message.ID).Errorw("failed to process Redis stream message", zap.Error(err))
					el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
					continue
				}
				msgsToAcknowledge = append(msgsToAcknowledge, message.ID)
			}

			if len(msgsToAcknowledge) == 0 {
				continue
			}

			// Even if acknowledging fails, since we handled the message, we are good to proceed.
			if err := client.XAck(ctx, entry.Stream, consumersGroup, msgsToAcknowledge...).Err(); err != nil {
				log.With("stream", entry.Stream, "message_ids", msgsToAcknowledge).Errorw("failed to acknowledge messages from the Redis stream", zap.Error(err))
			}
			if streamToLastEntryMapping[entry.Stream] != ">" {
				streamToLastEntryMapping[entry.Stream] = msgsToAcknowledge[len(msgsToAcknowledge)-1]
			}
		}
		updateReadGroupArgs()
	}
}

func (el *EventListener) handleOne(stream string, message redis.XMessage, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
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
		return fmt.Errorf("failed to marshal the event data, rejecting the event, %w", err)
	}
	log.With("stream", stream).Info("dispatching the event on the data channel...")
	if err = dispatch(eventBody); err != nil {
		return fmt.Errorf("failed dispatch a Redis stream event, %w", err)
	}
	return nil
}
