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
)

// ValidateEventSource validates gateway event source
func (ese *FileEventSourceExecutor) ValidateEventSource(ctx context.Context, es *gateways.EventSource) (*gateways.ValidEventSource, error) {
	v := &gateways.ValidEventSource{}
	fwc, err := parseEventSource(es.Data)
	if err != nil {
		gateways.SetValidEventSource(v, fmt.Sprintf("%s. err: %s", gateways.ErrEventSourceParseFailed, err.Error()), false)
		return v, nil
	}
	if err = validateFileWatcher(fwc); err != nil {
		gateways.SetValidEventSource(v, err.Error(), false)
		return v, gateways.ErrInvalidEventSource
	}
	gateways.SetValidEventSource(v, "", true)
	return v, nil
}

func validateFileWatcher(fwc *fileWatcher) error {
	if fwc == nil {
		return fmt.Errorf("configuration must be non empty")
	}
	if fwc.Type == "" {
		return fmt.Errorf("type must be specified")
	}
	if fwc.Directory == "" {
		return fmt.Errorf("directory must be specified")
	}
	if fwc.Path == "" {
		return fmt.Errorf("path must be specified")
	}
	return nil
}
