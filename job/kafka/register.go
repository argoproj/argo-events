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
	"github.com/blackrock/axis/job"
	"go.uber.org/zap"
)

const (
	// StreamTypeKafka defines the exact string representation for Kafka stream types
	StreamTypeKafka = "KAFKA"
	topicKey        = "topic"
	partitionKey    = "partition"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) (job.Signal, error) {
	abstract.Log.Info("creating signal", zap.String("type", abstract.Stream.Type))
	consumer, err := sarama.NewConsumer([]string{abstract.Stream.URL}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kafka cluster at %s", abstract.Stream.URL)
	}
	k := &kafka{
		AbstractSignal: abstract,
		stop:           make(chan struct{}),
		consumer:       consumer,
	}
	var ok bool
	if k.topic, ok = abstract.Stream.Attributes[topicKey]; !ok {
		return nil, fmt.Errorf(job.ErrMissingAttribute, abstract.Stream.Type, topicKey)
	}
	var pString string
	if pString, ok = abstract.Stream.Attributes[partitionKey]; !ok {
		return nil, fmt.Errorf(job.ErrMissingAttribute, abstract.Stream.Type, partitionKey)
	}
	pInt, err := strconv.ParseInt(pString, 10, 32)
	if err != nil {
		return nil, err
	}
	k.partition = int32(pInt)

	return k, nil
}

// Kafka will be added to the executor session
func Kafka(es *job.ExecutorSession) {
	es.AddStreamFactory(StreamTypeKafka, &factory{})
}
