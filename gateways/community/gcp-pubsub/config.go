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

package pubsub

import (
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

const ArgoEventsEventSourceVersion = "v0.11"

// GcpPubSubEventSourceExecutor implements Eventing
type GcpPubSubEventSourceExecutor struct {
	Log *logrus.Logger
}

// pubSubEventSource contains configuration to subscribe to GCP PubSub topic
type pubSubEventSource struct {
	// ProjectID is the unique identifier for your project on GCP
	ProjectID string `json:"projectID"`
	// TopicProjectID identifies the project where the topic should exist or be created
	// (assumed to be the same as ProjectID by default)
	TopicProjectID string `json:"topicProjectID"`
	// Topic on which a subscription will be created
	Topic string `json:"topic"`
	// CredentialsFile is the file that contains credentials to authenticate for GCP
	CredentialsFile string `json:"credentialsFile"`
}

func parseEventSource(es string) (interface{}, error) {
	var n *pubSubEventSource
	err := yaml.Unmarshal([]byte(es), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
