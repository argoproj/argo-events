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

package nats

import (
	"github.com/argoproj/argo-events/common"
	"github.com/ghodss/yaml"
	natslib "github.com/nats-io/go-nats"
	"k8s.io/apimachinery/pkg/util/wait"
)

const ArgoEventsEventSourceVersion = "v0.10"

// NatsEventSourceExecutor implements Eventing
type NatsEventSourceExecutor struct {
	Log *common.ArgoEventsLogger
}

// Nats contains configuration to connect to NATS cluster
type natsConfig struct {
	// URL to connect to natsConfig cluster
	URL string `json:"url"`
	// Subject name
	Subject string `json:"subject"`
	// Backoff holds parameters applied to connection.
	Backoff *wait.Backoff `json:"backoff,omitempty"`
	// conn represents a bare connection to a nats-server.
	conn *natslib.Conn
}

func parseEventSource(es string) (interface{}, error) {
	var n *natsConfig
	err := yaml.Unmarshal([]byte(es), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
