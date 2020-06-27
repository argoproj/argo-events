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
	"encoding/json"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventListener implements Eventing kafka event source
type EventListener struct {
	// Logger logs stuff
	Logger *logrus.Logger
}

func verifyPartitionAvailable(part int32, partitions []int32) bool {
	for _, p := range partitions {
		if part == p {
			return true
		}
	}
	return false
}

// StartEventSource starts an event source
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

func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, channels *server.Channels) error {
	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var kafkaEventSource *v1alpha1.KafkaEventSource
	if err := yaml.Unmarshal(eventSource.Value, &kafkaEventSource); err != nil {
		return errors.Wrapf(err, "failed to parse event source %s", eventSource.Name)
	}

	var consumer sarama.Consumer

	logger.Infoln("connecting to Kafka cluster...")
	if err := server.Connect(common.GetConnectionBackoff(kafkaEventSource.ConnectionBackoff), func() error {
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
		return errors.Wrapf(err, "failed to connect to Kafka broker for event source %s", eventSource.Name)
	}

	logger = logger.WithField("partition-id", kafkaEventSource.Partition)

	logger.Infoln("parsing the partition value...")
	pInt, err := strconv.ParseInt(kafkaEventSource.Partition, 10, 32)
	if err != nil {
		return errors.Wrapf(err, "failed to parse Kafka partition %s for event source %s", kafkaEventSource.Partition, eventSource.Name)
	}
	partition := int32(pInt)

	logger.Infoln("getting available partitions...")
	availablePartitions, err := consumer.Partitions(kafkaEventSource.Topic)
	if err != nil {
		return errors.Wrapf(err, "failed to get the available partitions for topic %s and event source %s", kafkaEventSource.Topic, eventSource.Name)
	}

	logger.Infoln("verifying the partition exists within available partitions...")
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		return errors.Wrapf(err, "partition %d is not available. event source %s", partition, eventSource.Name)
	}

	logger.Infoln("getting partition consumer...")
	partitionConsumer, err := consumer.ConsumePartition(kafkaEventSource.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		return errors.Wrapf(err, "failed to create consumer partition for event source %s", eventSource.Name)
	}

	logger.Info("listening to messages on the partition...")
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			logger.Infoln("dispatching event on the data channel...")
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
				logger.WithError(err).Errorln("failed to marshal the event data, rejecting the event...")
				continue
			}
			channels.Data <- eventBody

		case err := <-partitionConsumer.Errors():
			return errors.Wrapf(err, "failed to consume messages for event source %s", eventSource.Name)

		case <-channels.Done:
			logger.Infoln("event source is stopped, closing partition consumer")
			err = partitionConsumer.Close()
			if err != nil {
				logger.WithError(err).Error("failed to close consumer")
			}
			return nil
		}
	}
}
