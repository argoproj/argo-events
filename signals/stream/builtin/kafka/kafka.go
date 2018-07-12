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
	"log"
	"strconv"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	topicKey     = "topic"
	partitionKey = "partition"
	EventType    = "org.apache.kafka.pub"
)

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
type kafka struct{}

// New creates a new kafka signaler
func New() sdk.Listener {
	return new(kafka)
}

func (*kafka) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	// parse out the attributes
	topic, partition, err := parseAttributes(signal.Stream.Attributes)
	if err != nil {
		return nil, err
	}

	consumer, err := sarama.NewConsumer([]string{signal.Stream.URL}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster at %s", signal.Stream.URL)
	}

	availablePartitions, err := consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("unable to get available partitions for kafka topic '%s'. cause: %s", topic, err.Error())
	}
	if ok := verifyPartitionAvailable(partition, availablePartitions); !ok {
		return nil, fmt.Errorf("partition %v does not exist for topic '%s'", partition, topic)
	}

	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		return nil, fmt.Errorf("failed to create partition consumer for topic '%s' and partition: %v. cause: %s", topic, partition, err.Error())
	}

	events := make(chan *v1alpha1.Event)

	// start listening for messages
	go func() {
		defer close(events)
		for {
			select {
			case msg := <-partitionConsumer.Messages():
				event := &v1alpha1.Event{
					Context: v1alpha1.EventContext{
						EventID:            fmt.Sprintf("partition-%v-offset-%v", msg.Partition, msg.Offset),
						EventType:          EventType,
						EventTime:          metav1.Time{Time: msg.Timestamp},
						CloudEventsVersion: sdk.CloudEventsVersion,
						Extensions:         make(map[string]string),
					},
					Data: msg.Value,
				}
				log.Printf("signal '%s' received msg", signal.Name)
				events <- event
			case err := <-partitionConsumer.Errors():
				event := &v1alpha1.Event{
					Context: v1alpha1.EventContext{
						EventID:            fmt.Sprintf("partition-%v-", err.Partition),
						EventType:          EventType,
						CloudEventsVersion: sdk.CloudEventsVersion,
						Extensions:         map[string]string{sdk.ContextExtensionErrorKey: err.Err.Error()},
					},
				}
				log.Printf("signal '%s' received error", signal.Name)
				events <- event
			case <-done:
				if err := partitionConsumer.Close(); err != nil {
					log.Printf("failed to close partition consumer for signal '%s': %s", signal.Name, err)
				}
				if err := consumer.Close(); err != nil {
					log.Panicf("failed to close consumer for signal '%s': %s", signal.Name, err)
				}
				log.Printf("shut down signal '%s'", signal.Name)
				return
			}
		}
	}()

	return events, nil
}

func parseAttributes(attr map[string]string) (string, int32, error) {
	topic, ok := attr["topic"]
	if !ok {
		return "", 0, sdk.ErrMissingRequiredAttribute
	}
	var pString string
	if pString, ok = attr[partitionKey]; !ok {
		return topic, 0, fmt.Errorf(sdk.ErrMissingAttribute, partitionKey)
	}
	pInt, err := strconv.ParseInt(pString, 10, 32)
	if err != nil {
		return topic, 0, err
	}
	partition := int32(pInt)
	return topic, partition, nil
}

func verifyPartitionAvailable(part int32, partitions []int32) bool {
	for _, p := range partitions {
		if part == p {
			return true
		}
	}
	return false
}
