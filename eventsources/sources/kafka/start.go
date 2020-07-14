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
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/pkg/errors"
)

// EventListener implements Eventing kafka event source
type EventListener struct {
	EventSourceName  string
	EventName        string
	KafkaEventSource v1alpha1.KafkaEventSource
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
func (el *EventListener) StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).WithFields(map[string]interface{}{
		logging.LabelEventSourceType: el.GetEventSourceType(),
		logging.LabelEventSourceName: el.GetEventSourceName(),
		logging.LabelEventName:       el.GetEventName(),
	})
	defer sources.Recover(el.GetEventName())

	log.Infoln("start kafka event source...")
	kafkaEventSource := &el.KafkaEventSource

	var consumer sarama.Consumer

	log.Infoln("connecting to Kafka cluster...")
	if err := sources.Connect(common.GetConnectionBackoff(kafkaEventSource.ConnectionBackoff), func() error {
		var err error
		config := sarama.NewConfig()

		if kafkaEventSource.TLS != nil {
			tlsConfig, err := common.GetTLSConfig(kafkaEventSource.TLS.CACertPath, kafkaEventSource.TLS.ClientCertPath, kafkaEventSource.TLS.ClientKeyPath)
			if err != nil {
				return errors.Wrap(err, "failed to get the tls configuration")
			}
			config.Net.TLS.Config = tlsConfig
			config.Net.TLS.Enable = true
		} else {
			consumer, err = sarama.NewConsumer([]string{kafkaEventSource.URL}, nil)
			if err != nil {
				return err
			}
		}

		consumer, err = sarama.NewConsumer([]string{kafkaEventSource.URL}, config)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to Kafka broker for event source %s", el.GetEventName())
	}

	log = log.WithField("partition-id", kafkaEventSource.Partition)

	log.Infoln("parsing the partition value...")
	pInt, err := strconv.ParseInt(kafkaEventSource.Partition, 10, 32)
	if err != nil {
		return errors.Wrapf(err, "failed to parse Kafka partition %s for event source %s", kafkaEventSource.Partition, el.GetEventName())
	}
	partition := int32(pInt)

	log.Infoln("getting available partitions...")
	availablePartitions, err := consumer.Partitions(kafkaEventSource.Topic)
	if err != nil {
		return errors.Wrapf(err, "failed to get the available partitions for topic %s and event source %s", kafkaEventSource.Topic, el.GetEventName())
	}

	log.Infoln("verifying the partition exists within available partitions...")
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		return errors.Wrapf(err, "partition %d is not available. event source %s", partition, el.GetEventName())
	}

	log.Infoln("getting partition consumer...")
	partitionConsumer, err := consumer.ConsumePartition(kafkaEventSource.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		return errors.Wrapf(err, "failed to create consumer partition for event source %s", el.GetEventName())
	}

	log.Info("listening to messages on the partition...")
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			log.Infoln("dispatching event on the data channel...")
			eventData := &events.KafkaEventData{
				Topic:     msg.Topic,
				Partition: int(msg.Partition),
				Timestamp: msg.Timestamp.String(),
			}
			if kafkaEventSource.JSONBody {
				eventData.Body = (*json.RawMessage)(&msg.Value)
			} else {
				eventData.Body = msg.Value
			}
			eventBody, err := json.Marshal(eventData)
			if err != nil {
				log.WithError(err).Errorln("failed to marshal the event data, rejecting the event...")
				continue
			}
			if err = dispatch(eventBody); err != nil {
				log.WithError(err).Errorln("failed to dispatch event...")
			}

		case err := <-partitionConsumer.Errors():
			return errors.Wrapf(err, "failed to consume messages for event source %s", el.GetEventName())

		case <-stopCh:
			log.Infoln("event source is stopped, closing partition consumer")
			err = partitionConsumer.Close()
			if err != nil {
				log.WithError(err).Error("failed to close consumer")
			}
			return nil
		}
	}
}
