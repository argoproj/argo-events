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
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/server"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
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

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go listener.listenEvents(eventSource, dataCh, errorCh, doneCh)

	return server.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, listener.Logger)
}

func (listener *EventListener) listenEvents(eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer server.Recover(eventSource.Name)

	logger := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	logger.Infoln("parsing the event source...")
	var kafkaEventSource *v1alpha1.KafkaEventSource
	if err := yaml.Unmarshal(eventSource.Value, kafkaEventSource); err != nil {
		errorCh <- err
		return
	}

	var consumer sarama.Consumer

	logger.Infoln("connecting to Kafka cluster...")
	if err := server.Connect(&wait.Backoff{
		Steps:    kafkaEventSource.ConnectionBackoff.Steps,
		Jitter:   kafkaEventSource.ConnectionBackoff.Jitter,
		Duration: kafkaEventSource.ConnectionBackoff.Duration,
		Factor:   kafkaEventSource.ConnectionBackoff.Factor,
	}, func() error {
		var err error
		consumer, err = sarama.NewConsumer([]string{kafkaEventSource.URL}, nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		logger.WithError(err).WithField(common.LabelURL, kafkaEventSource.URL).Error("failed to connect")
		errorCh <- err
		return
	}

	logger = logger.WithField("partition-id", kafkaEventSource.Partition)

	logger.Infoln("parsing the partition value...")
	pInt, err := strconv.ParseInt(kafkaEventSource.Partition, 10, 32)
	if err != nil {
		errorCh <- err
		return
	}
	partition := int32(pInt)

	logger.Infoln("getting available partitions...")
	availablePartitions, err := consumer.Partitions(kafkaEventSource.Topic)
	if err != nil {
		errorCh <- err
		return
	}

	logger.Infoln("verifying the partition exists within available partitions...")
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		errorCh <- errors.Errorf("partition %d is not available", partition)
		return
	}

	logger.Infoln("getting partition consumer...")
	partitionConsumer, err := consumer.ConsumePartition(kafkaEventSource.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		errorCh <- err
		return
	}

	logger.Info("listening to messages on the partition...")
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			logger.Infoln("dispatching event on the data channel...")
			dataCh <- msg.Value

		case err := <-partitionConsumer.Errors():
			errorCh <- err
			return

		case <-doneCh:
			err = partitionConsumer.Close()
			if err != nil {
				logger.WithError(err).Error("failed to close consumer")
			}
			return
		}
	}
}
