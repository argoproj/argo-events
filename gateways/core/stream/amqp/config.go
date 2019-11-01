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

package amqp

import (
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	amqplib "github.com/streadway/amqp"
)

const ArgoEventsEventSourceVersion = "v0.10"

// EventListener implements Eventing for amqp event source
type EventListener struct {
	Log *logrus.Logger
}

// amqp contains configuration required to connect to rabbitmq service and process messages
type amqp struct {
	// eventSource contains configuration to connect to amqp service to consume messages
	eventSource *v1alpha1.AMQPEventSource
	// Connection manages the serialization and deserialization of frames from IO
	// and dispatches the frames to the appropriate channel.
	conn *amqplib.Connection
}

func parseEventSource(eventSource string) (interface{}, error) {
	var a *amqp
	err := yaml.Unmarshal([]byte(eventSource), &a)
	if err != nil {
		return nil, err
	}
	return a, nil
}
