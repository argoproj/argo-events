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
	"context"
	"fmt"

	"github.com/argoproj/argo-events/gateways"
)

// ValidateEventSource validates the gateway event source
func (ese *KafkaEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	kafkaConfig, err := parseEventSource(es.Data)
	if err != nil {
		return v, gateways.ErrEventSourceParseFailed
	}
	if kafkaConfig == nil {
		return v, fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidEventSource)
	}
	if kafkaConfig.URL == "" {
		return v, fmt.Errorf("%+v, url must be specified", gateways.ErrInvalidEventSource)
	}
	if kafkaConfig.Topic == "" {
		return v, fmt.Errorf("%+v, topic must be specified", gateways.ErrInvalidEventSource)
	}
	if kafkaConfig.Partition == "" {
		return v, fmt.Errorf("%+v, partition must be specified", gateways.ErrInvalidEventSource)
	}
	return v, nil
}
