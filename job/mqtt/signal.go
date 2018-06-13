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
	"time"

	"github.com/blackrock/axis/job"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

type mqtt struct {
	job.AbstractSignal
	client MQTTlib.Client
	stop   chan struct{}
	msgCh  chan MQTTlib.Message
}

func (m *mqtt) Start(events chan job.Event) error {
	opts := MQTTlib.NewClientOptions().AddBroker(m.MQTT.URL).SetClientID(m.Name)
	m.client = MQTTlib.NewClient(opts)
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		m.Log.Warn("failed to connect to mqtt client", zap.String("url", m.MQTT.URL))
		return token.Error()
	}

	// subscribe to the topic
	if token := m.client.Subscribe(m.MQTT.Topic, 0, m.handleMsg); token.Wait() && token.Error() != nil {
		m.Log.Warn("failed to subscribe to mqtt topic", zap.String("topic", m.MQTT.Topic))
		return token.Error()
	}
	go m.listen(events)
	return nil
}

func (m *mqtt) Stop() error {
	defer close(m.msgCh)
	m.stop <- struct{}{}
	if token := m.client.Unsubscribe(m.MQTT.Topic); token.Wait() && token.Error() != nil {
		m.Log.Warn("failed to unsubscribe from mqtt", zap.String("topic", m.MQTT.Topic))
		return token.Error()
	}
	m.client.Disconnect(250)
	return nil
}

// callback message handler passes the message to the mqtt message channel
func (m *mqtt) handleMsg(client MQTTlib.Client, message MQTTlib.Message) {
	if !message.Duplicate() {
		m.msgCh <- message
	}
}

func (m *mqtt) listen(events chan job.Event) {
	for {
		select {
		case msg := <-m.msgCh:
			event := &event{
				mqtt:      m,
				msg:       msg,
				timestamp: time.Now().UTC(),
			}
			// perform constraint checks
			ok := m.CheckConstraints(event.GetTimestamp())
			if !ok {
				event.SetError(job.ErrFailedTimeConstraint)
			}
			m.Log.Debug("sending mqtt event", zap.String("nodeID", event.GetID()))
			events <- event
		case <-m.stop:
			return
		}
	}
}
