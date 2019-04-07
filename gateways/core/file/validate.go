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

package file

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gateway event source
func (ese *FileEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	return gwcommon.ValidateGatewayEventSource(es, ArgoEventsEventSourceVersion, parseEventSource, validateFileWatcher)
}

func validateFileWatcher(config interface{}) error {
	fwc := config.(*fileWatcher)
	if fwc == nil {
		return gwcommon.ErrNilEventSource
	}
	if fwc.Type == "" {
		return fmt.Errorf("type must be specified")
	}
	err := fwc.WatchPathConfig.Validate()
	return err
}
