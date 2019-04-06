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
	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/wait"
)

const ArgoEventsEventSourceVersion = "v0.10"

// KafkaEventSourceExecutor implements Eventing
type KafkaEventSourceExecutor struct {
	Log *common.ArgoEventsLogger
}

// kafka defines configuration required to connect to kafka cluster
type kafka struct {
	// URL to kafka cluster
	URL string `json:"url"`
	// Partition name
	Partition string `json:"partition"`
	// Topic name
	Topic string `json:"topic"`
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
	// Consumer manages PartitionConsumers which process Kafka messages from brokers.
	consumer sarama.Consumer
}

func parseEventSource(eventSource string) (interface{}, error) {
	var n *kafka
	err := yaml.Unmarshal([]byte(eventSource), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
