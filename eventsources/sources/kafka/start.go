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

package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing kafka event source
type EventListener struct {
	EventSourceName  string
	EventName        string
	KafkaEventSource v1alpha1.KafkaEventSource
	Metrics          *metrics.Metrics
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
	return apicommon.KafkaEvent
}

func verifyPartitionAvailable(part int32, partitions []int32) bool {
	for _, p := range partitions {
		if part == p {
			return true
		}
	}
	return false
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	log.Info("start kafka event source...")
	kafkaEventSource := &el.KafkaEventSource

	if kafkaEventSource.ConsumerGroup == nil {
		return el.partitionConsumer(ctx, log, kafkaEventSource, dispatch)
	} else {
		return el.consumerGroupConsumer(ctx, log, kafkaEventSource, dispatch)
	}
}

func (el *EventListener) consumerGroupConsumer(ctx context.Context, log *zap.SugaredLogger, kafkaEventSource *v1alpha1.KafkaEventSource, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	config, err := getSaramaConfig(kafkaEventSource, log)
	if err != nil {
		return err
	}

	switch kafkaEventSource.ConsumerGroup.RebalanceStrategy {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		log.Info("Invalid rebalance strategy, using default: range")
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	}

	consumer := Consumer{
		ready:            make(chan bool),
		dispatch:         dispatch,
		logger:           log,
		kafkaEventSource: kafkaEventSource,
		eventSourceName:  el.EventSourceName,
		eventName:        el.EventName,
		metrics:          el.Metrics,
	}

	urls := strings.Split(kafkaEventSource.URL, ",")
	client, err := sarama.NewConsumerGroup(urls, kafkaEventSource.ConsumerGroup.GroupName, config)
	if err != nil {
		log.Errorf("Error creating consumer group client: %v", err)
		return err
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, []string{kafkaEventSource.Topic}, &consumer); err != nil {
				log.Errorf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				log.Infof("Error from context: %v", ctx.Err())
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	log.Info("Sarama consumer group up and running!...")

	<-ctx.Done()
	log.Info("terminating: context cancelled")
	wg.Wait()

	if err = client.Close(); err != nil {
		log.Errorf("Error closing client: %v", err)
		return err
	}

	return nil
}

func (el *EventListener) partitionConsumer(ctx context.Context, log *zap.SugaredLogger, kafkaEventSource *v1alpha1.KafkaEventSource, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	defer sources.Recover(el.GetEventName())

	log.Info("start kafka event source...")

	var consumer sarama.Consumer

	log.Info("connecting to Kafka cluster...")
	if err := common.DoWithRetry(kafkaEventSource.ConnectionBackoff, func() error {
		var err error

		config, err := getSaramaConfig(kafkaEventSource, log)
		if err != nil {
			return err
		}

		urls := strings.Split(kafkaEventSource.URL, ",")
		consumer, err = sarama.NewConsumer(urls, config)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to connect to Kafka broker for event source %s, %w", el.GetEventName(), err)
	}

	log = log.With("partition-id", kafkaEventSource.Partition)

	log.Info("parsing the partition value...")
	pInt, err := strconv.ParseInt(kafkaEventSource.Partition, 10, 32)
	if err != nil {
		return fmt.Errorf("failed to parse Kafka partition %s for event source %s, %w", kafkaEventSource.Partition, el.GetEventName(), err)
	}
	partition := int32(pInt)

	log.Info("getting available partitions...")
	availablePartitions, err := consumer.Partitions(kafkaEventSource.Topic)
	if err != nil {
		return fmt.Errorf("failed to get the available partitions for topic %s and event source %s, %w", kafkaEventSource.Topic, el.GetEventName(), err)
	}

	log.Info("verifying the partition exists within available partitions...")
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		return fmt.Errorf("partition %d is not available. event source %s, %w", partition, el.GetEventName(), err)
	}

	log.Info("getting partition consumer...")
	partitionConsumer, err := consumer.ConsumePartition(kafkaEventSource.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		return fmt.Errorf("failed to create consumer partition for event source %s, %w", el.GetEventName(), err)
	}

	processOne := func(msg *sarama.ConsumerMessage) error {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		log.Info("dispatching event on the data channel...")
		eventData := &events.KafkaEventData{
			Topic:     msg.Topic,
			Key:       string(msg.Key),
			Partition: int(msg.Partition),
			Timestamp: msg.Timestamp.String(),
			Metadata:  kafkaEventSource.Metadata,
		}

		headers := make(map[string]string)

		for _, recordHeader := range msg.Headers {
			headers[string(recordHeader.Key)] = string(recordHeader.Value)
		}

		eventData.Headers = headers

		if kafkaEventSource.JSONBody {
			eventData.Body = (*json.RawMessage)(&msg.Value)
		} else {
			eventData.Body = msg.Value
		}
		eventBody, err := json.Marshal(eventData)
		if err != nil {
			return fmt.Errorf("failed to marshal the event data, rejecting the event, %w", err)
		}

		kafkaID := genUniqueID(el.GetEventSourceName(), el.GetEventName(), kafkaEventSource.URL, msg.Topic, msg.Partition, msg.Offset)

		if err = dispatch(eventBody, eventsourcecommon.WithID(kafkaID)); err != nil {
			return fmt.Errorf("failed to dispatch a Kafka event, %w", err)
		}
		return nil
	}

	log.Info("listening to messages on the partition...")
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			if err := processOne(msg); err != nil {
				log.Errorw("failed to process a Kafka message", zap.Error(err))
				el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			}
		case err := <-partitionConsumer.Errors():
			return fmt.Errorf("failed to consume messages for event source %s, %w", el.GetEventName(), err)

		case <-ctx.Done():
			log.Info("event source is stopped, closing partition consumer")
			err = partitionConsumer.Close()
			if err != nil {
				log.Errorw("failed to close consumer", zap.Error(err))
			}
			return nil
		}
	}
}

func getSaramaConfig(kafkaEventSource *v1alpha1.KafkaEventSource, log *zap.SugaredLogger) (*sarama.Config, error) {
	config, err := common.GetSaramaConfigFromYAMLString(kafkaEventSource.Config)
	if err != nil {
		return nil, err
	}

	if kafkaEventSource.Version == "" {
		config.Version = sarama.V1_0_0_0
	} else {
		version, err := sarama.ParseKafkaVersion(kafkaEventSource.Version)
		if err != nil {
			log.Errorf("Error parsing Kafka version: %v", err)
			return nil, err
		}
		config.Version = version
	}

	if kafkaEventSource.SASL != nil {
		config.Net.SASL.Enable = true

		config.Net.SASL.Mechanism = sarama.SASLMechanism(kafkaEventSource.SASL.GetMechanism())
		if config.Net.SASL.Mechanism == "SCRAM-SHA-512" {
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA512New} }
		} else if config.Net.SASL.Mechanism == "SCRAM-SHA-256" {
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA256New} }
		}

		user, err := common.GetSecretFromVolume(kafkaEventSource.SASL.UserSecret)
		if err != nil {
			log.Errorf("Error getting user value from secret: %v", err)
			return nil, err
		}
		config.Net.SASL.User = user

		password, err := common.GetSecretFromVolume(kafkaEventSource.SASL.PasswordSecret)
		if err != nil {
			log.Errorf("Error getting password value from secret: %v", err)
			return nil, err
		}
		config.Net.SASL.Password = password
	}

	if kafkaEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(kafkaEventSource.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}

	if kafkaEventSource.ConsumerGroup != nil {
		if kafkaEventSource.ConsumerGroup.Oldest {
			config.Consumer.Offsets.Initial = sarama.OffsetOldest
		}
	}
	return config, nil
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ready            chan bool
	dispatch         func([]byte, ...eventsourcecommon.Option) error
	logger           *zap.SugaredLogger
	kafkaEventSource *v1alpha1.KafkaEventSource
	eventSourceName  string
	eventName        string
	metrics          *metrics.Metrics
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

var (
	eventBusErr *eventbuscommon.EventBusError
)

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		if err := consumer.processOne(session, message); err != nil {
			consumer.metrics.EventProcessingFailed(consumer.eventSourceName, consumer.eventName)
			if errors.As(err, &eventBusErr) { // EventBus error, do not continue.
				consumer.logger.Errorw("failed to process a Kafka message due to event bus issue", zap.Error(err))
				break
			} else {
				consumer.logger.Errorw("failed to process a Kafka message, skip it", zap.Error(err))
				continue
			}
		}
		if consumer.kafkaEventSource.LimitEventsPerSecond > 0 {
			// 1000000000 is 1 second in nanoseconds
			d := (1000000000 / time.Duration(consumer.kafkaEventSource.LimitEventsPerSecond) * time.Nanosecond) * time.Nanosecond
			consumer.logger.Debugf("Sleeping for: %v.", d)
			time.Sleep(d)
		}
	}

	return nil
}

func (consumer *Consumer) processOne(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) error {
	defer func(start time.Time) {
		consumer.metrics.EventProcessingDuration(consumer.eventSourceName, consumer.eventName, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	consumer.logger.Info("dispatching event on the data channel...")
	eventData := &events.KafkaEventData{
		Topic:     message.Topic,
		Key:       string(message.Key),
		Partition: int(message.Partition),
		Timestamp: message.Timestamp.String(),
		Metadata:  consumer.kafkaEventSource.Metadata,
	}

	headers := make(map[string]string)

	for _, recordHeader := range message.Headers {
		headers[string(recordHeader.Key)] = string(recordHeader.Value)
	}

	eventData.Headers = headers

	if consumer.kafkaEventSource.JSONBody {
		eventData.Body = (*json.RawMessage)(&message.Value)
	} else {
		eventData.Body = message.Value
	}
	eventBody, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal the event data, rejecting the event, %w", err)
	}

	messageID := genUniqueID(consumer.eventSourceName, consumer.eventName, consumer.kafkaEventSource.URL, message.Topic, message.Partition, message.Offset)

	if err = consumer.dispatch(eventBody, eventsourcecommon.WithID(messageID)); err != nil {
		return fmt.Errorf("failed to dispatch a kafka event, %w", err)
	}
	session.MarkMessage(message, "")
	return nil
}

// Function can be passed as Option to generate unique id for kafka event
// eventSourceName:eventName:kafka-url:topic:partition:offset
func genUniqueID(eventSourceName, eventName, kafkaURL, topic string, partition int32, offset int64) string {
	kafkaID := fmt.Sprintf("%s:%s:%s:%s:%d:%d", eventSourceName, eventName, strings.Split(kafkaURL, ",")[0], topic, partition, offset)

	return kafkaID
}
