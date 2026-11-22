/*
Copyright 2026 The Argoproj Authors.

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

package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
)

const (
	testBackoff     = "10ms"
	testEventSource = "test-es"
	testEventName   = "test-event"
)

// newWSTestServer runs a minimal websocket server that invokes handler on
// every accepted connection.
func newWSTestServer(t *testing.T, handler func(*websocket.Conn)) string {
	t.Helper()
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestStartListeningDispatchesMessages(t *testing.T) {
	url := newWSTestServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"hello":"world"}`))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"n":2}`))
		// keep the connection open until the client goes away
		_, _, _ = conn.ReadMessage()
	})

	el := &EventListener{
		EventSourceName: testEventSource,
		EventName:       testEventName,
		WebSocketEventSource: v1alpha1.WebSocketEventSource{
			URL:      url,
			JSONBody: true,
			Metadata: map[string]string{"foo": "bar"},
			ConnectionBackoff: &v1alpha1.Backoff{
				Duration: &v1alpha1.Int64OrString{Type: v1alpha1.String, StrVal: testBackoff},
				Steps:    3,
			},
		},
		Metrics: metrics.NewMetrics("argo-events"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var got [][]byte
	done := make(chan error, 1)
	go func() {
		done <- el.StartListening(ctx, func(body []byte, _ ...eventsourcecommon.Option) error {
			mu.Lock()
			got = append(got, body)
			n := len(got)
			mu.Unlock()
			if n == 2 {
				cancel()
			}
			return nil
		})
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("StartListening did not return in time")
	}

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, got, 2)

	var first events.WebSocketEventData
	require.NoError(t, json.Unmarshal(got[0], &first))
	assert.Equal(t, map[string]string{"foo": "bar"}, first.Metadata)
	firstJSON, err := json.Marshal(first.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"hello":"world"}`, string(firstJSON))

	var second events.WebSocketEventData
	require.NoError(t, json.Unmarshal(got[1], &second))
	secondJSON, err := json.Marshal(second.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"n":2}`, string(secondJSON))
}

func TestStartListeningPlainBodyIsBase64(t *testing.T) {
	// with JSONBody=false the raw payload bytes are marshalled as base64,
	// matching the other byte-oriented event sources (mqtt, nats, ...)
	url := newWSTestServer(t, func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("hello"))
		_, _, _ = conn.ReadMessage()
	})

	el := &EventListener{
		EventSourceName: testEventSource,
		EventName:       testEventName,
		WebSocketEventSource: v1alpha1.WebSocketEventSource{
			URL: url,
			ConnectionBackoff: &v1alpha1.Backoff{
				Duration: &v1alpha1.Int64OrString{Type: v1alpha1.String, StrVal: testBackoff},
				Steps:    3,
			},
		},
		Metrics: metrics.NewMetrics("argo-events"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	got := make(chan []byte, 1)
	go func() {
		done <- el.StartListening(ctx, func(body []byte, _ ...eventsourcecommon.Option) error {
			select {
			case got <- body:
				cancel()
			default:
			}
			return nil
		})
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("StartListening did not return in time")
	}

	var eventData events.WebSocketEventData
	require.NoError(t, json.Unmarshal(<-got, &eventData))
	bodyJSON, err := json.Marshal(eventData.Body)
	require.NoError(t, err)
	assert.Equal(t, `"aGVsbG8="`, string(bodyJSON)) // base64("hello")
}

func TestStartListeningReconnectsAfterServerDrop(t *testing.T) {
	var mu sync.Mutex
	connections := 0
	url := newWSTestServer(t, func(conn *websocket.Conn) {
		mu.Lock()
		connections++
		n := connections
		mu.Unlock()
		// first connection: send one message then drop the connection
		if n == 1 {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("first"))
			return
		}
		// second connection: send another message and stay open
		_ = conn.WriteMessage(websocket.TextMessage, []byte("second"))
		_, _, _ = conn.ReadMessage()
	})

	el := &EventListener{
		EventSourceName: testEventSource,
		EventName:       testEventName,
		WebSocketEventSource: v1alpha1.WebSocketEventSource{
			URL: url,
			ConnectionBackoff: &v1alpha1.Backoff{
				Duration: &v1alpha1.Int64OrString{Type: v1alpha1.String, StrVal: testBackoff},
				Steps:    5,
			},
		},
		Metrics: metrics.NewMetrics("argo-events"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu2 sync.Mutex
	var bodies []string
	done := make(chan error, 1)
	go func() {
		done <- el.StartListening(ctx, func(body []byte, _ ...eventsourcecommon.Option) error {
			var eventData events.WebSocketEventData
			if err := json.Unmarshal(body, &eventData); err == nil {
				b, _ := json.Marshal(eventData.Body)
				mu2.Lock()
				bodies = append(bodies, string(b))
				n := len(bodies)
				mu2.Unlock()
				if n == 2 {
					cancel()
				}
			}
			return nil
		})
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(15 * time.Second):
		t.Fatal("StartListening did not reconnect in time")
	}

	mu2.Lock()
	defer mu2.Unlock()
	require.Equal(t, []string{`"Zmlyc3Q="`, `"c2Vjb25k"`}, bodies) // base64("first"), base64("second")
}

func TestStartListeningFailsOnBadURL(t *testing.T) {
	el := &EventListener{
		EventSourceName: testEventSource,
		EventName:       testEventName,
		WebSocketEventSource: v1alpha1.WebSocketEventSource{
			URL: "ws://127.0.0.1:1/unreachable",
			ConnectionBackoff: &v1alpha1.Backoff{
				Duration: &v1alpha1.Int64OrString{Type: v1alpha1.String, StrVal: "5ms"},
				Steps:    2,
			},
		},
		Metrics: metrics.NewMetrics("argo-events"),
	}

	// DoWithRetry exhausts the backoff steps and returns an error, which
	// surfaces as a reconnection attempt; cancel via context after a moment.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := el.StartListening(ctx, func([]byte, ...eventsourcecommon.Option) error { return nil })
	assert.NoError(t, err) // context cancellation is a clean shutdown
}

func TestReconnectDelay(t *testing.T) {
	assert.Equal(t, time.Second, reconnectDelay(nil))
	assert.Equal(t, time.Second, reconnectDelay(&v1alpha1.Backoff{}))
	assert.Equal(t, 2*time.Second, reconnectDelay(&v1alpha1.Backoff{
		Duration: &v1alpha1.Int64OrString{Type: v1alpha1.String, StrVal: "2s"},
	}))
	assert.Equal(t, 500*time.Millisecond, reconnectDelay(&v1alpha1.Backoff{
		Duration: &v1alpha1.Int64OrString{Type: v1alpha1.Int64, Int64Val: int64(500 * time.Millisecond)},
	}))
	// invalid duration falls back to the default
	assert.Equal(t, time.Second, reconnectDelay(&v1alpha1.Backoff{
		Duration: &v1alpha1.Int64OrString{Type: v1alpha1.String, StrVal: "not-a-duration"},
	}))
}
