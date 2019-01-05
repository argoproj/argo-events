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
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
)

// StartEventSource starts an event source
func (ese *MqttEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", *eventSource.Name).Msg("operating on event source")
	m, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(m, eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

func (ese *MqttEventSourceExecutor) listenEvents(m *mqtt, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		dataCh <- msg.Payload()
	}
	opts := MQTTlib.NewClientOptions().AddBroker(m.URL).SetClientID(m.ClientId)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		errorCh <- token.Error()
		return
	}
	if token := client.Subscribe(m.Topic, 0, handler); token.Wait() && token.Error() != nil {
		errorCh <- token.Error()
		return
	}

	<-doneCh
	token := client.Unsubscribe(m.Topic)
	if token.Error() != nil {
		// nothing to do
		ese.Log.Error().Err(token.Error()).Str("event-source-name", *eventSource.Name).Msg("failed to unsubscribe client")
	}
}
