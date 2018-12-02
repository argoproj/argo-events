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

package aws_sns

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

func (ce *AWSSNSConfigExecutor) Validate(config *gateways.ConfigContext) error {
	snsConfig, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if snsConfig.Port == "" {
		return fmt.Errorf("port is not provided")
	}
	if snsConfig.Endpoint == "" {
		return fmt.Errorf("endpoint not provided")
	}
	return nil
}
