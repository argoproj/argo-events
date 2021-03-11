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
	"context"
	"encoding/json"
	"time"

	natslib "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	metrics "github.com/argoproj/argo-events/metrics"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for nats event source
type EventListener struct {
	EventSourceName string
	EventName       string
	NATSEventSource v1alpha1.NATSEventsSource
	Metrics         *metrics.Metrics
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
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.NATSEvent
}

// StartListening starts listening events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	log := logging.FromContext(ctx).
		With(logging.LabelEventSourceType, el.GetEventSourceType(), logging.LabelEventName, el.GetEventName())
	defer sources.Recover(el.GetEventName())

	natsEventSource := &el.NATSEventSource

	var opt []natslib.Option
	if natsEventSource.TLS != nil {
		tlsConfig, err := common.GetTLSConfig(natsEventSource.TLS)
		if err != nil {
			return errors.Wrap(err, "failed to get the tls configuration")
		}
		opt = append(opt, natslib.Secure(tlsConfig))
	}

	if natsEventSource.Auth != nil {
		switch {
		case natsEventSource.Auth.Basic != nil:
			username, err := common.GetSecretFromVolume(natsEventSource.Auth.Basic.Username)
			if err != nil {
				return err
			}
			password, err := common.GetSecretFromVolume(natsEventSource.Auth.Basic.Password)
			if err != nil {
				return err
			}
			opt = append(opt, natslib.UserInfo(username, password))
		case natsEventSource.Auth.Token != nil:
			token, err := common.GetSecretFromVolume(natsEventSource.Auth.Token)
			if err != nil {
				return err
			}
			opt = append(opt, natslib.Token(token))
		case natsEventSource.Auth.NKey != nil:
			nkeyFile, err := common.GetSecretVolumePath(natsEventSource.Auth.NKey)
			if err != nil {
				return err
			}
			o, err := natslib.NkeyOptionFromSeed(nkeyFile)
			if err != nil {
				return errors.Wrap(err, "failed to get NKey")
			}
			opt = append(opt, o)
		case natsEventSource.Auth.Credential != nil:
			cFile, err := common.GetSecretVolumePath(natsEventSource.Auth.Credential)
			if err != nil {
				return err
			}
			opt = append(opt, natslib.UserCredentials(cFile))
		}
	}

	var conn *natslib.Conn
	log.Info("connecting to nats cluster...")
	if err := common.Connect(natsEventSource.ConnectionBackoff, func() error {
		var err error
		if conn, err = natslib.Connect(natsEventSource.URL, opt...); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "failed to connect to the nats server for event source %s", el.GetEventName())
	}
	defer conn.Close()

	if natsEventSource.JSONBody {
		log.Info("assuming all events have a json body...")
	}

	log.Info("subscribing to messages on the queue...")
	_, err := conn.Subscribe(natsEventSource.Subject, func(msg *natslib.Msg) {
		defer func(start time.Time) {
			el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
		}(time.Now())

		eventData := &events.NATSEventData{
			Subject:  msg.Subject,
			Metadata: natsEventSource.Metadata,
		}
		if natsEventSource.JSONBody {
			eventData.Body = (*json.RawMessage)(&msg.Data)
		} else {
			eventData.Body = msg.Data
		}

		eventBody, err := json.Marshal(eventData)
		if err != nil {
			log.Errorw("failed to marshal the event data, rejecting the event...", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
			return
		}
		log.Info("dispatching the event on data channel...")
		if err = dispatch(eventBody); err != nil {
			log.Errorw("failed to dispatch a NATS event", zap.Error(err))
			el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
		}
	})
	if err != nil {
		return errors.Wrapf(err, "failed to subscribe to the subject %s for event source %s", natsEventSource.Subject, el.GetEventName())
	}

	conn.Flush()
	if err := conn.LastError(); err != nil {
		return errors.Wrapf(err, "connection failure for event source %s", el.GetEventName())
	}

	<-ctx.Done()
	log.Info("event source is stopped")
	return nil
}
