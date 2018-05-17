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
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"go.uber.org/zap"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) job.Signal {
	abstract.Log.Info("creating signal", zap.String("raw", abstract.Kafka.String()))
	consumer, err := sarama.NewConsumer([]string{abstract.Kafka.URL}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to connect to kafka cluster at %s", abstract.Kafka.URL))
	}
	return &kafka{
		AbstractSignal: abstract,
		stop:           make(chan struct{}),
		consumer:       consumer,
	}
}

// Kafka will be added to the executor session
func Kafka(es *job.ExecutorSession) {
	es.AddFactory(v1alpha1.SignalTypeKafka, &factory{})
}
