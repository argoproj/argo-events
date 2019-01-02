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
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	"strings"
)

// Validate validates gateway configuration
func (sgce *StorageGridConfigExecutor) Validate(config *gateways.EventSourceContext) error {
	sg, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if sg == nil {
		return gateways.ErrEmptyConfig
	}
	if sg.Port == "" {
		return fmt.Errorf("%+v, must specify port", gateways.ErrInvalidConfig)
	}
	if sg.Endpoint == "" {
		return fmt.Errorf("%+v, must specify endpoint", gateways.ErrInvalidConfig)
	}
	if !strings.HasPrefix(sg.Endpoint, "/") {
		return fmt.Errorf("%+v, endpoint must start with '/'", gateways.ErrInvalidConfig)
	}
	return nil
}
