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

	"github.com/Shopify/sarama"
	"github.com/blackrock/axis/job"
	"go.uber.org/zap"
)

type kafka struct {
	job.AbstractSignal
	consumer          sarama.Consumer
	partitionConsumer sarama.PartitionConsumer
	stop              chan struct{}
}

func (k *kafka) Start(events chan job.Event) error {
	var err error
	availablePartitions, err := k.consumer.Partitions(k.Kafka.Topic)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to get available partitions for kafka topic '%s'. cause: %s", k.Kafka.Topic, err.Error()))
	}

	if ok := k.verifyPartitionAvailable(availablePartitions); !ok {
		return fmt.Errorf(fmt.Sprintf("partition %v does not exist for topic '%s'", k.Kafka.Partition, k.Kafka.Topic))
	}

	k.partitionConsumer, err = k.consumer.ConsumePartition(k.Kafka.Topic, k.Kafka.Partition, sarama.OffsetNewest)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to create partition consumer for topic '%s' and partition: %v. cause: %s", k.Kafka.Topic, k.Kafka.Partition, err.Error()))
	}

	go k.listen(events)
	return nil
}

func (k *kafka) Stop() error {
	k.stop <- struct{}{}
	err := k.partitionConsumer.Close()
	if err != nil {
		return err
	}
	err = k.consumer.Close()
	if err != nil {
		return err
	}
	return nil
}

func (k *kafka) listen(events chan job.Event) {
	for {
		select {
		case msg := <-k.partitionConsumer.Messages():
			event := &event{
				kafka:    k,
				kafkaMsg: msg,
			}
			// perform constraint checks
			ok := k.CheckConstraints(event.GetTimestamp())
			if !ok {
				event.SetError(job.ErrFailedTimeConstraint)
			}
			k.Log.Debug("sending kafka event", zap.String("nodeID", event.GetID()))
			events <- event
		case err := <-k.partitionConsumer.Errors():
			event := &event{
				kafka:    k,
				kafkaMsg: nil,
			}
			event.SetError(err.Err)
			k.Log.Debug("sending kafka error", zap.String("nodeID", event.GetID()), zap.Error(err.Err))
			events <- event
		case <-k.stop:
			return
		}
	}
}

func (k *kafka) verifyPartitionAvailable(partitions []int32) bool {
	for _, p := range partitions {
		if k.Kafka.Partition == p {
			return true
		}
	}
	return false
}
