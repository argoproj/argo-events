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
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
)

// KafkaConfigExecutor implements ConfigExecutor
type KafkaConfigExecutor struct {
	*gateways.GatewayConfig
}

// kafka defines configuration required to connect to kafka cluster
// +k8s:openapi-gen=true
type kafka struct {
	// URL to kafka cluster
	URL string `json:"url"`

	// Partition name
	Partition string `json:"partition"`

	// Topic name
	Topic string `json:"topic"`
}

func parseEventSource(eventSource *string) (*kafka, error) {
	var n *kafka
	err := yaml.Unmarshal([]byte(*eventSource), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
