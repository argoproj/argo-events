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
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"k8s.io/apimachinery/pkg/util/wait"
)

func verifyPartitionAvailable(part int32, partitions []int32) bool {
	for _, p := range partitions {
		if part == p {
			return true
		}
	}
	return false
}

// StartEventSource starts an event source
func (ese *KafkaEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)

	log.Info("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*kafka), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, ese.Log)
}

func (ese *KafkaEventSourceExecutor) listenEvents(k *kafka, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)

	if err := gateways.Connect(&wait.Backoff{
		Steps:    k.Backoff.Steps,
		Jitter:   k.Backoff.Jitter,
		Duration: k.Backoff.Duration,
		Factor:   k.Backoff.Factor,
	}, func() error {
		var err error
		k.consumer, err = sarama.NewConsumer([]string{k.URL}, nil)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.WithError(err).WithField(common.LabelURL, k.URL).Error("failed to connect")
		errorCh <- err
		return
	}

	pInt, err := strconv.ParseInt(k.Partition, 10, 32)
	if err != nil {
		errorCh <- err
		return
	}
	partition := int32(pInt)

	availablePartitions, err := k.consumer.Partitions(k.Topic)
	if err != nil {
		errorCh <- err
		return
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		errorCh <- fmt.Errorf("partition %d is not available", partition)
		return
	}

	partitionConsumer, err := k.consumer.ConsumePartition(k.Topic, partition, sarama.OffsetNewest)
	if err != nil {
		errorCh <- err
		return
	}

	log.Info("starting to subscribe to messages")
	for {
		select {
		case msg := <-partitionConsumer.Messages():
			dataCh <- msg.Value

		case err := <-partitionConsumer.Errors():
			errorCh <- err
			return

		case <-doneCh:
			err = partitionConsumer.Close()
			if err != nil {
				log.WithError(err).Error("failed to close consumer")
			}
			return
		}
	}
}
