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

package hdfs

import (
	"context"
	"errors"
	"time"

	"github.com/argoproj/argo-events/gateways/common/fileevent"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gateway event source
func (ese *EventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(es.Data, parseEventSource, validateGatewayConfig)
}

func validateGatewayConfig(config interface{}) error {
	gwc := config.(*GatewayConfig)
	if gwc == nil {
		return gwcommon.ErrNilEventSource
	}
	if gwc.Type == "" {
		return errors.New("type is required")
	}
	op := fileevent.NewOp(gwc.Type)
	if op == 0 {
		return errors.New("type is invalid")
	}
	if gwc.CheckInterval != "" {
		_, err := time.ParseDuration(gwc.CheckInterval)
		if err != nil {
			return errors.New("failed to parse interval")
		}
	}
	err := gwc.WatchPathConfig.Validate()
	if err != nil {
		return err
	}
	err = gwc.GatewayClientConfig.Validate()
	return err
}
