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

package aws_sqs

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

func (ce *AWSSQSConfigExecutor) Validate(config *gateways.ConfigContext) error {
	sqsConfig, err := parseConfig(config.Data.Config)
	if err != nil {
		return gateways.ErrConfigParseFailed
	}
	if sqsConfig.Queue == "" {
		return fmt.Errorf("queue name not provided")
	}
	if sqsConfig.Region == "" {
		return fmt.Errorf("region not provided")
	}
	if sqsConfig.SecretKey == nil {
		return fmt.Errorf("secret key not provided")
	}
	if sqsConfig.AccessKey == nil {
		return fmt.Errorf("access key not provided")
	}
	if sqsConfig.Frequency == "" {
		return fmt.Errorf("frequency to try receive message is not provided")
	}
	return nil
}

