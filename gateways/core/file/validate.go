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
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

// Validate validates gateway configuration
func (fw *FileWatcherConfigExecutor) Validate(config *gateways.EventSourceContext) error {
	fwc, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if fwc == nil {
		return fmt.Errorf("%+v, configuration must be non empty", gateways.ErrInvalidConfig)
	}
	if fwc.Type == "" {
		return fmt.Errorf("%+v, type must be specified", gateways.ErrInvalidConfig)
	}
	if fwc.Directory == "" {
		return fmt.Errorf("%+v, directory must be specified", gateways.ErrInvalidConfig)
	}
	if fwc.Path == "" {
		return fmt.Errorf("%+v, path must be specified", gateways.ErrInvalidConfig)
	}
	return nil
}
