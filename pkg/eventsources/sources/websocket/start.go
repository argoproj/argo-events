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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for websocket event source
type EventListener struct {
	EventSourceName      string
	EventName            string
	WebSocketEventSource v1alpha1.WebSocketEventSource
	Metrics              *metrics.Metrics
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.WebSocketEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	log.Info("starting WebSocket event source...")
	wsEventSource := &el.WebSocketEventSource

	if wsEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	for {
		err := el.connectAndListen(ctx, dispatch, log)
		if ctx.Err() != nil {
			log.Info("event source is stopped, closing the websocket connection...")
			return nil
		}
		log.Errorw("websocket connection lost, reconnecting...", zap.Error(err))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(reconnectDelay(wsEventSource.ConnectionBackoff)):
		}
	}
}

// connectAndListen dials the WebSocket server and dispatches every incoming
// message until the connection breaks or the context is canceled.
func (el *EventListener) connectAndListen(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) error {
	wsEventSource := &el.WebSocketEventSource

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	if wsEventSource.TLS != nil {
		tlsConfig, err := sharedutil.GetTLSConfig(wsEventSource.TLS)
		if err != nil {
			return fmt.Errorf("failed to get the tls configuration, %w", err)
		}
		dialer.TLSClientConfig = tlsConfig
	} else {
		// make sure wss:// URLs still verify the server certificate by default
		dialer.TLSClientConfig = &tls.Config{} //nolint:gosec // empty config uses the default verification
	}

	header := http.Header{}
	for k, v := range wsEventSource.Headers {
		header.Set(k, v)
	}
	if wsEventSource.Auth != nil {
		username, err := sharedutil.GetSecretFromVolume(wsEventSource.Auth.Username)
		if err != nil {
			return fmt.Errorf("username not found, %w", err)
		}
		password, err := sharedutil.GetSecretFromVolume(wsEventSource.Auth.Password)
		if err != nil {
			return fmt.Errorf("password not found, %w", err)
		}
		header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	}

	log.Infow("connecting to the websocket server...", "url", wsEventSource.URL)
	var conn *websocket.Conn
	if err := sharedutil.DoWithRetry(wsEventSource.ConnectionBackoff, func() error {
		c, resp, err := dialer.DialContext(ctx, wsEventSource.URL, header)
		if err != nil {
			if resp != nil {
				return fmt.Errorf("failed to connect to %s: %w (status: %s)", wsEventSource.URL, err, resp.Status)
			}
			return fmt.Errorf("failed to connect to %s: %w", wsEventSource.URL, err)
		}
		conn = c
		return nil
	}); err != nil {
		return fmt.Errorf("failed to connect to the websocket server for event source %s, %w", el.GetEventName(), err)
	}
	defer conn.Close()
	// closing the connection on context cancellation unblocks ReadMessage
	readDone := make(chan struct{})
	defer close(readDone)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-readDone:
		}
	}()

	log.Info("connected, listening for events...")
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}
		// control frames (ping/pong/close) are handled by the reader itself;
		// only dispatch data frames
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		el.handleMessage(payload, dispatch, log)
	}
}

// handleMessage marshals a single websocket message and dispatches it.
func (el *EventListener) handleMessage(payload []byte, dispatch func([]byte, ...eventsourcecommon.Option) error, log *zap.SugaredLogger) {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	wsEventSource := &el.WebSocketEventSource
	eventData := &events.WebSocketEventData{
		Body:     payload,
		Metadata: wsEventSource.Metadata,
	}
	if wsEventSource.JSONBody {
		eventData.Body = (*json.RawMessage)(&payload)
	}

	eventBody, err := json.Marshal(eventData)
	if err != nil {
		log.Errorw("failed to marshal the event data, rejecting the event...", zap.Error(err))
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		return
	}
	log.Info("dispatching event on the data channel...")
	if err = dispatch(eventBody); err != nil {
		log.Errorw("failed to dispatch WebSocket event...", zap.Error(err))
		el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
	}
}

// reconnectDelay returns the delay before the next reconnection attempt. It
// defaults to the first step of the configured backoff (1s).
func reconnectDelay(backoff *v1alpha1.Backoff) time.Duration {
	if backoff == nil || backoff.Duration == nil {
		return time.Second
	}
	if backoff.Duration.Type == v1alpha1.Int64 {
		return time.Duration(backoff.Duration.Int64Value())
	}
	d, err := time.ParseDuration(backoff.Duration.StrVal)
	if err != nil || d <= 0 {
		return time.Second
	}
	return d
}
