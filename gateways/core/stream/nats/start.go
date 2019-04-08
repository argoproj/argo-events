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
	"github.com/argoproj/argo-events/gateways"
	natslib "github.com/nats-io/go-nats"
	"k8s.io/apimachinery/pkg/util/wait"
)

// StartEventSource starts an event source
func (ese *NatsEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	log := ese.Log.WithField(common.LabelEventSource, eventSource.Name)
	log.Info("operating on event source")

	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}

	dataCh := make(chan []byte)
	errorCh := make(chan error)
	doneCh := make(chan struct{}, 1)

	go ese.listenEvents(config.(*natsConfig), eventSource, dataCh, errorCh, doneCh)

	return gateways.HandleEventsFromEventSource(eventSource.Name, eventStream, dataCh, errorCh, doneCh, ese.Log)
}

func (ese *NatsEventSourceExecutor) listenEvents(n *natsConfig, eventSource *gateways.EventSource, dataCh chan []byte, errorCh chan error, doneCh chan struct{}) {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithFields(
		map[string]interface{}{
			common.LabelEventSource: eventSource.Name,
			common.LabelURL:         n.URL,
			"subject":               n.Subject,
		},
	)

	if err := gateways.Connect(&wait.Backoff{
		Steps:    n.Backoff.Steps,
		Jitter:   n.Backoff.Jitter,
		Duration: n.Backoff.Duration,
		Factor:   n.Backoff.Factor,
	}, func() error {
		var err error
		if n.conn, err = natslib.Connect(n.URL); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.WithError(err).Error("connection failed")
		errorCh <- err
		return
	}

	log.Info("subscribing to messages")
	_, err := n.conn.Subscribe(n.Subject, func(msg *natslib.Msg) {
		dataCh <- msg.Data
	})
	if err != nil {
		log.WithError(err).Error("failed to subscribe")
		errorCh <- err
		return
	}
	n.conn.Flush()
	if err := n.conn.LastError(); err != nil {
		errorCh <- err
		return
	}

	<-doneCh
}
