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
	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const ArgoEventsEventSourceVersion = "v0.10"

// MqttEventSourceExecutor implements Eventing
type MqttEventSourceExecutor struct {
	Log *logrus.Logger
}

// mqtt contains information to connect to MQTT broker
type mqtt struct {
	// URL to connect to broker
	URL string `json:"url"`
	// Topic name
	Topic string `json:"topic"`
	// Client ID
	ClientId string `json:"clientId"`
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
	// It is an MQTT client for communicating with an MQTT server
	client mqttlib.Client
}

func parseEventSource(eventSource string) (interface{}, error) {
	var m *mqtt
	err := yaml.Unmarshal([]byte(eventSource), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
