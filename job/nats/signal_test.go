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
	"strconv"
	"testing"
	"time"

	"github.com/argoproj/argo-events/job"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/gnatsd/test"
	natsio "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNats(t *testing.T) {
	natsEmbeddedServerOpts := server.Options{
		Host:           "localhost",
		Port:           4222,
		NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
	}
	es := job.New(nil, nil, zap.NewNop())
	NATS(es)
	natsFactory, ok := es.GetStreamFactory(StreamTypeNats)
	assert.True(t, ok, "nats factory is not found")
	abstractSignal := job.AbstractSignal{
		Signal: v1alpha1.Signal{
			Name: "nats-test",
			Stream: &v1alpha1.Stream{
				Type:       "NATS",
				URL:        "nats://" + natsEmbeddedServerOpts.Host + ":" + strconv.Itoa(natsEmbeddedServerOpts.Port),
				Attributes: map[string]string{"subject": "test"},
			},
		},
		Log:     zap.NewNop(),
		Session: es,
	}
	signal, err := natsFactory.Create(abstractSignal)
	assert.Nil(t, err)
	testCh := make(chan job.Event)

	// attempt to start signal with no gnats server running
	err = signal.Start(testCh)
	assert.NotNil(t, err)

	// run an embedded gnats server
	testServer := test.RunServer(&natsEmbeddedServerOpts)
	defer testServer.Shutdown()
	err = signal.Start(testCh)
	assert.Nil(t, err)

	// now publish a NATS message on same subject
	conn, err := natsio.Connect(abstractSignal.Signal.Stream.URL)
	defer conn.Close()
	assert.Nil(t, err)
	err = conn.Publish("test", []byte("hello, world"))
	assert.Nil(t, err)

	// now get the event on the channel
	nextMsg, ok := <-testCh
	assert.True(t, ok, "next NATS message not found on event")
	assert.Equal(t, "", nextMsg.GetID())
	assert.Equal(t, []byte("hello, world"), nextMsg.GetBody())
	assert.Equal(t, "test", nextMsg.GetSource())
	assert.True(t, time.Now().After(nextMsg.GetTimestamp()), "NATS message timestamp before now")
	assert.Equal(t, signal, nextMsg.GetSignal())

	err = signal.Stop()
	assert.Nil(t, err)
}
