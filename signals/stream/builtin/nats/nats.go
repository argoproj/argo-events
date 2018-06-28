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

package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
	plugin "github.com/hashicorp/go-plugin"
	natsio "github.com/nats-io/go-nats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	subjectKey = "subject"
	EventType  = "com.github.nats-io.pub"
)

// nats is a plugin for a stream signal
type nats struct {
	natsConn         *natsio.Conn
	natsSubscription *natsio.Subscription
	msgCh            chan *natsio.Msg
	stop             chan struct{}
}

// New creates a new nats signaler
func New() shared.Signaler {
	return &nats{
		msgCh: make(chan *natsio.Msg),
		stop:  make(chan struct{}),
	}
}

// Start nats signal
func (n *nats) Start(signal *v1alpha1.Signal) (<-chan *v1alpha1.Event, error) {
	// parse out the attributes
	subject, ok := signal.Stream.Attributes[subjectKey]
	if !ok {
		return nil, shared.ErrMissingRequiredAttribute
	}
	var err error
	n.natsConn, err = natsio.Connect(signal.Stream.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats cluster url %s. Cause: %+v", signal.Stream.URL, err.Error())
	}
	n.natsSubscription, err = n.natsConn.ChanSubscribe(subject, n.msgCh)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to nats subject %s. Cause: %+v", subject, err.Error())
	}
	events := make(chan *v1alpha1.Event)
	go n.listen(events)
	return events, nil
}

// Stop nats signal
func (n *nats) Stop() error {
	defer n.natsConn.Close()
	defer close(n.msgCh)
	log.Printf("stopping signal")
	n.stop <- struct{}{}
	return n.natsSubscription.Unsubscribe()
}

func (n *nats) listen(events chan *v1alpha1.Event) {
	defer close(events)
	id := 0
	for {
		select {
		case natsMsg := <-n.msgCh:
			event := &v1alpha1.Event{
				Context: v1alpha1.EventContext{
					EventType:          EventType,
					CloudEventsVersion: shared.CloudEventsVersion,
					EventID:            natsMsg.Subject + "-" + strconv.Itoa(id),
					EventTime:          metav1.Time{Time: time.Now().UTC()},
					Extensions:         make(map[string]string),
				},
				Data: natsMsg.Data,
			}
			log.Printf("sending nat event")
			events <- event
		case <-n.stop:
			return
		}
	}
}

func main() {
	nats := New()
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			shared.SignalPluginName: shared.NewPlugin(nats),
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
