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

package mqtt

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
)

// MqttConfigExecutor implements ConfigExecutor
type MqttConfigExecutor struct {
	*gateways.GatewayConfig
}

// mqtt contains information to connect to MQTT broker
// +k8s:openapi-gen=true
type mqtt struct {
	// URL to connect to broker
	URL string `json:"url"`

	// Topic name
	Topic string `json:"topic"`

	// Client ID
	ClientId string `json:"clientId"`
}

func parseEventSource(eventSource *string) (*mqtt, error) {
	var m *mqtt
	err := yaml.Unmarshal([]byte(*eventSource), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
