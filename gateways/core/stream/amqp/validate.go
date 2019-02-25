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

package amqp

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gateway event source
func (ese *AMQPEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(es.Data, parseEventSource, validateAMQP)
}

func validateAMQP(config interface{}) error {
	a := config.(*amqp)
	if a == nil {
		return gwcommon.ErrNilEventSource
	}
	if a.URL == "" {
		return fmt.Errorf("url must be specified")
	}
	if a.RoutingKey == "" {
		return fmt.Errorf("routing key must be specified")
	}
	if a.ExchangeName == "" {
		return fmt.Errorf("exchange name must be specified")
	}
	if a.ExchangeType == "" {
		return fmt.Errorf("exchange type must be specified")
	}
	return nil
}
