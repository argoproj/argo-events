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

package main

import (
	"fmt"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
	plugin "github.com/hashicorp/go-plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	topicKey     = "topic"
	partitionKey = "partition"
	EventType    = "org.apache.kafka.pub"
)

type kafka struct {
	consumer          sarama.Consumer
	partitionConsumer sarama.PartitionConsumer
	stop              chan struct{}
}

// New creates a new kafka signaler
func New() shared.Signaler {
	return &kafka{
		stop: make(chan struct{}),
	}
}

func (k *kafka) Start(signal *v1alpha1.Signal) (<-chan *v1alpha1.Event, error) {
	// parse out the attributes
	topic, ok := signal.Stream.Attributes["topic"]
	if !ok {
		return nil, shared.ErrMissingRequiredAttribute
	}
	var pString string
	if pString, ok = signal.Stream.Attributes[partitionKey]; !ok {
		return nil, fmt.Errorf(shared.ErrMissingAttribute, partitionKey)
	}
	pInt, err := strconv.ParseInt(pString, 10, 32)
	if err != nil {
		return nil, err
	}
	partition := int32(pInt)

	k.consumer, err = sarama.NewConsumer([]string{signal.Stream.URL}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kafka cluster at %s", signal.Stream.URL)
	}

	availablePartitions, err := k.consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("unable to get available partitions for kafka topic '%s'. cause: %s", topic, err.Error())
	}

	if ok := k.verifyPartitionAvailable(partition, availablePartitions); !ok {
		return nil, fmt.Errorf("partition %v does not exist for topic '%s'", partition, topic)
	}

	k.partitionConsumer, err = k.consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		return nil, fmt.Errorf("failed to create partition consumer for topic '%s' and partition: %v. cause: %s", topic, partition, err.Error())
	}
	events := make(chan *v1alpha1.Event)
	go k.listen(events)
	return events, nil
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

func (k *kafka) listen(events chan *v1alpha1.Event) {
	defer close(events)
	for {
		select {
		case msg := <-k.partitionConsumer.Messages():
			event := &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					EventID:            fmt.Sprintf("partition-%v-offset-%v", msg.Partition, msg.Offset),
					EventType:          EventType,
					EventTime:          metav1.Time{Time: msg.Timestamp},
					CloudEventsVersion: shared.CloudEventsVersion,
					Extensions:         make(map[string]string),
				},
				Data: msg.Value,
			}
			events <- event
		case err := <-k.partitionConsumer.Errors():
			event := &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					EventID:            fmt.Sprintf("partition-%v-", err.Partition),
					EventType:          EventType,
					CloudEventsVersion: shared.CloudEventsVersion,
					Extensions:         map[string]string{shared.ContextExtensionErrorKey: err.Err.Error()},
				},
			}
			events <- event
		case <-k.stop:
			return
		}
	}
}

func (k *kafka) verifyPartitionAvailable(part int32, partitions []int32) bool {
	for _, p := range partitions {
		if part == p {
			return true
		}
	}
	return false
}

func main() {
	kafka := New()
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			shared.SignalPluginName: shared.NewPlugin(kafka),
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
