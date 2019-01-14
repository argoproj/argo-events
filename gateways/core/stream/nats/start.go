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
	"github.com/argoproj/argo-events/gateways"
	"github.com/nats-io/go-nats"
)

// StartEventSource starts an event source
func (ese *NatsEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	n, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(n, eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, &ese.Log)
}

func (ese *NatsEventSourceExecutor) listenEvents(n *natsConfig, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	nc, err := nats.Connect(n.URL)
	if err != nil {
		ese.Log.Error().Str("url", n.URL).Err(err).Msg("connection failed")
		errorCh <- err
		return
	}

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("starting to subscribe to messages")
	_, err = nc.Subscribe(n.Subject, func(msg *nats.Msg) {
		dataCh <- msg.Data
	})
	if err != nil {
		errorCh <- err
		return
	}
	nc.Flush()
	if err := nc.LastError(); err != nil {
		errorCh <- err
		return
	}

	<-doneCh
}
