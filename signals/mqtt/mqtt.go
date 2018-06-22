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
	"strconv"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	topicKey  = "topic"
	EventType = "mqtt.github.io.msg"
)

type mqtt struct {
	client MQTTlib.Client
	stop   chan struct{}
	msgCh  chan MQTTlib.Message

	//attribute fields
	topic string
}

// New creates a new mqtt signaler
func New() shared.Signaler {
	return &mqtt{
		msgCh: make(chan MQTTlib.Message),
		stop:  make(chan struct{}),
	}
}

func (m *mqtt) Start(signal *v1alpha1.Signal) (<-chan *v1alpha1.Event, error) {
	// parse out the attributes
	var ok bool
	m.topic, ok = signal.Stream.Attributes[topicKey]
	if !ok {
		return nil, shared.ErrMissingRequiredAttribute
	}

	opts := MQTTlib.NewClientOptions().AddBroker(signal.Stream.URL).SetClientID(signal.Name)
	m.client = MQTTlib.NewClient(opts)
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to mqtt client. cause: %s", token.Error())
	}

	// subscribe to the topic
	if token := m.client.Subscribe(m.topic, 0, m.handleMsg); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to subscribe to mqtt topic. cause: %s", token.Error())
	}
	events := make(chan *v1alpha1.Event)
	go m.listen(events)
	return events, nil
}

func (m *mqtt) Stop() error {
	defer close(m.msgCh)
	m.stop <- struct{}{}
	if token := m.client.Unsubscribe(m.topic); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from mqtt topic. cause: %s", token.Error())
	}
	m.client.Disconnect(0)
	return nil
}

// callback message handler passes the message to the mqtt message channel
func (m *mqtt) handleMsg(client MQTTlib.Client, message MQTTlib.Message) {
	if !message.Duplicate() {
		m.msgCh <- message
	}
}

func (m *mqtt) listen(events chan *v1alpha1.Event) {
	defer close(events)
	for {
		select {
		case msg := <-m.msgCh:
			event := &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					EventID:            strconv.FormatUint(uint64(msg.MessageID()), 16),
					EventType:          EventType,
					CloudEventsVersion: shared.CloudEventsVersion,
					EventTime:          metav1.Time{Time: time.Now().UTC()},
					Extensions:         make(map[string]string),
				},
				Data: msg.Payload(),
			}
			events <- event
		case <-m.stop:
			return
		}
	}
}
