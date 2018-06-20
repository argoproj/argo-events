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
	"fmt"

	"github.com/argoproj/argo-events/job"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

const (
	// StreamTypeMqtt defines the exact string representation for MQTT stream types
	StreamTypeMqtt = "MQTT"
	topicKey       = "topic"
)

type factory struct{}

func (f *factory) Create(abstract job.AbstractSignal) (job.Signal, error) {
	abstract.Log.Info("creating signal", zap.String("type", abstract.Stream.Type))
	m := &mqtt{
		AbstractSignal: abstract,
		stop:           make(chan struct{}),
		msgCh:          make(chan MQTTlib.Message),
	}
	var ok bool
	if m.topic, ok = abstract.Stream.Attributes[topicKey]; !ok {
		return nil, fmt.Errorf(job.ErrMissingAttribute, abstract.Stream.Type, topicKey)
	}
	return m, nil
}

// MQTT will be added to the executor session
func MQTT(es *job.ExecutorSession) {
	es.AddStreamFactory(StreamTypeMqtt, &factory{})
}
