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
	"strconv"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/gnatsd/test"
	natsio "github.com/nats-io/go-nats"
)

func TestSignal(t *testing.T) {
	natsEmbeddedServerOpts := server.Options{
		Host:           "localhost",
		Port:           4222,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
	}
	nats := New()

	signal := v1alpha1.Signal{
		Name: "nats-test",
		Stream: &v1alpha1.Stream{
			Type: "URL",
			URL:  "nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port),
		},
	}

	// start the signal - expect ErrMissingRequiredAttribute
	_, err := nats.Start(&signal)
	if err != shared.ErrMissingRequiredAttribute {
		t.Errorf("expected: %s\n found: %s", shared.ErrMissingRequiredAttribute, err)
	}

	// add required attributes
	subject := "test"
	signal.Stream.Attributes = map[string]string{"subject": subject}
	_, err = nats.Start(&signal)
	if err == nil {
		t.Errorf("expected: failed to connect to nats cluster\nfound: %s", err)
	}

	// run an embedded gnats server
	testServer := test.RunServer(&natsEmbeddedServerOpts)
	defer testServer.Shutdown()
	events, err := nats.Start(&signal)
	if err != nil {
		t.Error(err)
	}

	// publish a message
	conn, err := natsio.Connect(signal.Stream.URL)
	if err != nil {
		t.Fatalf("failed to connect to embedded nats server. cause: %s", err)
	}
	defer conn.Close()
	err = conn.Publish(subject, []byte("hello, world"))
	if err != nil {
		t.Fatalf("failed to publish test msg. cause: %s", err)
	}

	// now lets get the event
	nextMsg, ok := <-events
	if !ok {
		t.Errorf("failed to receive msg from events channel")
	}
	if nextMsg.Context.EventID != "test-0" {
		t.Errorf("event context eventID:\nexpected: %s\nactual: %s", "test-0", nextMsg.Context.EventID)
	}
	if nextMsg.Context.EventType != EventType {
		t.Errorf("event context EventType:\nexpected: %s\nactual: %s", EventType, nextMsg.Context.EventID)
	}
	if nextMsg.Context.CloudEventsVersion != shared.CloudEventsVersion {
		t.Errorf("event context CloudEventsVersion:\nexpected: %s\nactual: %s", shared.CloudEventsVersion, nextMsg.Context.CloudEventsVersion)
	}

	// stop the signal
	err = nats.Stop()
	if err != nil {
		t.Errorf("failed to stop signal. cause: %s", err)
	}

	// ensure events channel is closed
	if _, ok := <-events; ok {
		t.Errorf("expected read-only events channel to be closed after signal stop")
	}
}
