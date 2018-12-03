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

package google_pubsub

import (
	"fmt"
	"github.com/argoproj/argo-events/gateways"
)

func (ce *GCPPubSubConfigExecutor) Validate(config *gateways.ConfigContext) error {
	gps, err := parseConfig(config.Data.Config)
	if err != nil {
		ce.Log.Error().Err(err).Msg("failed to parse configuration")
		return err
	}
	if gps.ProjectId == "" {
		return fmt.Errorf("project id is not provided")
	}
	if gps.Subscription == "" {
		return fmt.Errorf("subscription is not provided")
	}
	if gps.Topic == "" {
		return fmt.Errorf("topic is not provided")
	}
	return nil
}
