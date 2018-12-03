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
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
)

// GCPPubSubConfigExecutor implements ConfigExecutor
type GCPPubSubConfigExecutor struct {
	*gateways.GatewayConfig
}

// GCPPubSubConfig contains information required to subscribe to a topic
// For more info, see https://cloud.google.com/pubsub/docs/subscriber
type GCPPubSubConfig struct {
	// Topic is the topic name to subscribe to
	Topic string `json:"topic"`

	// Subscription is subscription name
	Subscription string `json:"subscription"`

	// ProjectId is the ID of the project
	ProjectId string `json:"projectId"`
}

func parseConfig(config string) (*GCPPubSubConfig, error) {
	var gps *GCPPubSubConfig
	err := yaml.Unmarshal([]byte(config), &gps)
	if err != nil {
		return nil, err
	}
	return gps, nil
}
