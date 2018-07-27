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
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	natsio "github.com/nats-io/go-nats"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	subjectKey = "subject"
	EventType  = "com.github.nats-io.pub"
)

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
type nats struct{}

// New creates a new nats signaler
func New() sdk.Listener {
	return new(nats)
}

func (*nats) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	// parse out the attributes
	subject, ok := signal.Stream.Attributes[subjectKey]
	if !ok {
		return nil, sdk.ErrMissingRequiredAttribute
	}

	events := make(chan *v1alpha1.Event)

	var id uint64
	handler := func(msg *natsio.Msg) {
		event := &v1alpha1.Event{
			Context: v1alpha1.EventContext{
				EventType:          EventType,
				CloudEventsVersion: sdk.CloudEventsVersion,
				EventID:            msg.Subject + "-" + strconv.FormatUint(atomic.AddUint64(&id, 1), 10),
				EventTime:          metav1.Time{Time: time.Now().UTC()},
				Extensions:         make(map[string]string),
			},
			Data: msg.Data,
		}
		log.Printf("signal '%s' received msg", signal.Name)
		events <- event
	}

	conn, err := natsio.Connect(signal.Stream.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cluster url %s: %+v", signal.Stream.URL, err.Error())
	}
	sub, err := conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %+v", subject, err.Error())
	}

	// wait for done signal
	go func() {
		defer close(events)
		<-done
		del, _ := sub.Delivered()
		drop, _ := sub.Dropped()
		queue, _ := sub.QueuedMsgs()
		sub.Unsubscribe()
		conn.Close()
		log.Printf("shut down signal '%s'\nSubscription Stats:\nDelivered:%v\nDropped:%v\nQueued:%v", signal.Name, del, drop, queue)
	}()

	log.Printf("signal '%s' listening for NATS msgs on subject [%s]...", signal.Name, subject)
	return events, nil
}
