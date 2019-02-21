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

package storagegrid

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
)

// ValidateEventSource validates gateway event source
func (ese *StorageGridEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	sg, err := parseEventSource(es.Data)
	if err != nil {
		return &gateways.ValidEventSource{
			IsValid: false,
			Reason:  fmt.Sprintf("failed to parse event source. err: %+v", err),
		}, nil
	}
	if err = validateStorageGrid(sg); err != nil {
		return &gateways.ValidEventSource{
			Reason:  err.Error(),
			IsValid: false,
		}, nil
	}
	return &gateways.ValidEventSource{
		IsValid: true,
		Reason:  "valid",
	}, nil
}

func validateStorageGrid(sg *storageGrid) error {
	if sg == nil {
		return gateways.ErrEmptyEventSource
	}
	return gwcommon.ValidateWebhook(sg.Endpoint, sg.Port)
}
