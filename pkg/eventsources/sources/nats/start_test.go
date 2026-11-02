/*
Copyright 2018 The Argoproj Authors.

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
	"context"
	"net"
	"testing"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	"github.com/nats-io/nats-server/v2/server"
	natslib "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

const testSubject = "events.test"

// Reproduces an outage that outlasts the reconnect attempts: previously the listener kept waiting on the
// closed connection without receiving events or logging anything, it must now return so it is restarted.
func TestStartListening_ReturnsWhenReconnectsAreExhausted(t *testing.T) {
	shortenReconnectWait(t)
	natServer := runNatsServer(t, -1)
	url := natServer.ClientURL()

	maxReconnects := int32(2)
	dispatched, result := startListener(t, url, &maxReconnects)
	publishUntilDispatched(t, url, dispatched)

	stopNatsServer(natServer)

	select {
	case err := <-result:
		require.ErrorContains(t, err, "nats connection closed")
	case <-time.After(5 * time.Second):
		t.Fatal("listener kept waiting on a closed nats connection")
	}
}

func TestStartListening_ResubscribesAfterOutageWithUnlimitedReconnects(t *testing.T) {
	shortenReconnectWait(t)
	natServer := runNatsServer(t, -1)
	url := natServer.ClientURL()
	port := natServer.Addr().(*net.TCPAddr).Port

	unlimitedReconnects := int32(-1)
	dispatched, result := startListener(t, url, &unlimitedReconnects)
	publishUntilDispatched(t, url, dispatched)

	// Stay down for far more reconnect attempts than the limited test above allows.
	stopNatsServer(natServer)
	time.Sleep(time.Second)
	natServer = runNatsServer(t, port)
	defer stopNatsServer(natServer)

	publishUntilDispatched(t, url, dispatched)
	select {
	case err := <-result:
		t.Fatalf("listener stopped during the outage: %v", err)
	default:
	}
}

func runNatsServer(t *testing.T, port int) *server.Server {
	t.Helper()

	natsServer, err := server.NewServer(&server.Options{Host: "127.0.0.1", Port: port, NoLog: true, NoSigs: true})
	require.NoError(t, err)

	go natsServer.Start()

	natsReady := natsServer.ReadyForConnections(5 * time.Second)
	require.True(t, natsReady, "nats server did not start")

	return natsServer
}

func stopNatsServer(s *server.Server) {
	s.Shutdown()
	s.WaitForShutdown()
}

// startListener runs StartListening in the background and returns the
// dispatched events and the result.
func startListener(t *testing.T, url string, maxReconnects *int32) (<-chan []byte, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(logging.WithLogger(context.Background(), logging.NewArgoEventsLogger()))
	t.Cleanup(cancel)

	el := &EventListener{
		EventSourceName: "nats-test",
		EventName:       "nats-test",
		NATSEventSource: v1alpha1.NATSEventsSource{URL: url, Subject: testSubject, MaxReconnects: maxReconnects},
		Metrics:         metrics.NewMetrics("nats-test"),
	}
	dispatched := make(chan []byte, 100)
	errorResult := make(chan error, 1)
	go func() {
		testHandler := func(data []byte, _ ...eventsourcecommon.Option) error {
			dispatched <- data
			return nil
		}

		errorResult <- el.StartListening(ctx, testHandler)
	}()
	return dispatched, errorResult
}

// publishUntilDispatched publishes until the listener dispatches a message,
// as it may still be (re)subscribing.
func publishUntilDispatched(t *testing.T, url string, dispatched <-chan []byte) {
	t.Helper()
	conn, err := natslib.Connect(url)
	require.NoError(t, err)
	defer conn.Close()

	deadline := time.After(5 * time.Second)
	for {
		err := conn.Publish(testSubject, []byte("foo"))
		require.NoError(t, err)

		err = conn.Flush()
		require.NoError(t, err)

		select {
		case <-dispatched:
			return
		case <-time.After(100 * time.Millisecond):
		case <-deadline:
			t.Fatal("no event dispatched")
		}
	}
}

func shortenReconnectWait(t *testing.T) {
	original := defaultNatsReconnectWait
	defaultNatsReconnectWait = 50 * time.Millisecond
	t.Cleanup(func() { defaultNatsReconnectWait = original })
}
