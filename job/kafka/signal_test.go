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
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"

	"github.com/blackrock/axis/job"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"go.uber.org/zap"
)

func TestSignal(t *testing.T) {
	consumer := mocks.NewConsumer(t, sarama.NewConfig())
	consumer.SetTopicMetadata(map[string][]int32{"test": []int32{0}})
	expectedPartitionConsumer := consumer.ExpectConsumePartition("test", 0, -1)
	expectedPartitionConsumer.ExpectMessagesDrainedOnClose()
	expectedPartitionConsumer.ExpectErrorsDrainedOnClose()

	es := job.New(nil, nil, zap.NewNop())
	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "kafka-test",
			Stream: &v1alpha1.Stream{
				Type:       "KAFKA",
				URL:        "localhost",
				Attributes: map[string]string{"topic": "unknown"},
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	signal := &kafka{
		AbstractSignal: abstractSignal,
		consumer:       consumer,
		stop:           make(chan struct{}),
		topic:          "unknown",
		partition:      0,
	}
	testCh := make(chan job.Event)

	// unable to get available partitions
	err := signal.Start(testCh)
	assert.NotNil(t, err)

	// partition not available
	consumer.SetTopicMetadata(map[string][]int32{"unknown": []int32{1}})
	err = signal.Start(testCh)
	assert.NotNil(t, err)

	// success
	consumer.SetTopicMetadata(map[string][]int32{"test": []int32{0}})
	signal.topic = "test"
	signal.partition = 0

	err = signal.Start(testCh)
	assert.Nil(t, err)

	// send message
	testMsg := &sarama.ConsumerMessage{
		Topic:     "test",
		Partition: 0,
		Offset:    1,
		Timestamp: time.Now(),
		Key:       []byte("key"),
		Value:     []byte("hello, world"),
	}
	expectedPartitionConsumer.YieldMessage(testMsg)

	// verify the message
	event := <-testCh
	assert.Equal(t, "", event.GetID())
	assert.Equal(t, "test", event.GetSource())
	assert.Equal(t, []byte("hello, world"), event.GetBody())
	assert.Equal(t, signal, event.GetSignal())

	// send an error
	err = fmt.Errorf("this is a test error")
	expectedPartitionConsumer.YieldError(err)

	// verify the error
	event = <-testCh
	assert.Equal(t, err, event.GetError())

	err = signal.Stop()
	assert.Nil(t, err)
}
