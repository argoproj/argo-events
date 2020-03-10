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
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	cloudevents "github.com/cloudevents/sdk-go"
	cloudeventsnats "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/nats"
	"github.com/google/uuid"
)

// updateSubscriberClients updates the active clients for event subscribers
func (gatewayContext *GatewayContext) updateSubscriberClients() {
	if gatewayContext.gateway.Spec.Subscribers == nil {
		return
	}

	if gatewayContext.httpSubscribers == nil {
		gatewayContext.httpSubscribers = make(map[string]cloudevents.Client)
	}
	if gatewayContext.natsSubscribers == nil {
		gatewayContext.natsSubscribers = make(map[string]cloudevents.Client)
	}

	// http subscribers
	for _, subscriber := range gatewayContext.gateway.Spec.Subscribers.HTTP {
		if _, ok := gatewayContext.httpSubscribers[subscriber]; !ok {
			t, err := cloudevents.NewHTTPTransport(
				cloudevents.WithTarget(subscriber),
				cloudevents.WithEncoding(cloudevents.HTTPBinaryV03),
			)
			if err != nil {
				gatewayContext.logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to create a transport")
				continue
			}

			client, err := cloudevents.NewClient(t)
			if err != nil {
				gatewayContext.logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to create a client")
				continue
			}
			gatewayContext.logger.WithField("subscriber", subscriber).Infoln("added a client for the subscriber")
			gatewayContext.httpSubscribers[subscriber] = client
		}
	}

	// nats subscribers
	for _, subscriber := range gatewayContext.gateway.Spec.Subscribers.NATS {
		if _, ok := gatewayContext.natsSubscribers[subscriber.Name]; !ok {
			t, err := cloudeventsnats.New(subscriber.ServerURL, subscriber.Subject)
			if err != nil {
				gatewayContext.logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to create a transport")
				continue
			}

			client, err := cloudevents.NewClient(t)
			if err != nil {
				gatewayContext.logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to create a client")
				continue
			}
			gatewayContext.logger.WithField("subscriber", subscriber).Infoln("added a client for the subscriber")
			gatewayContext.natsSubscribers[subscriber.Name] = client
		}
	}
}

// dispatchEvent dispatches event to gateway transformer for further processing
func (gatewayContext *GatewayContext) dispatchEvent(gatewayEvent *gateways.Event) error {
	logger := gatewayContext.logger.WithField(common.LabelEventSource, gatewayEvent.Name)
	logger.Infoln("dispatching event to subscribers")

	if gatewayContext.gateway.Spec.Subscribers == nil {
		logger.Warnln("no active subscribers to send event to.")
		return nil
	}

	cloudEvent, err := gatewayContext.transformEvent(gatewayEvent)
	if err != nil {
		return err
	}

	completeSuccess := true

	// http subscribers
	for _, subscriber := range gatewayContext.gateway.Spec.Subscribers.HTTP {
		client, ok := gatewayContext.httpSubscribers[subscriber]
		if !ok {
			gatewayContext.logger.WithField("subscriber", subscriber).Warnln("unable to send event. no client found for the subscriber")
			completeSuccess = false
			continue
		}

		if _, _, err := client.Send(context.Background(), *cloudEvent); err != nil {
			logger.WithError(err).WithField("target", subscriber).Warnln("failed to send the event")
			completeSuccess = false
			continue
		}
	}

	// NATS subscribers
	for _, subscriber := range gatewayContext.gateway.Spec.Subscribers.NATS {
		client, ok := gatewayContext.natsSubscribers[subscriber.Name]
		if !ok {
			gatewayContext.logger.WithField("subscriber", subscriber).Warnln("unable to send event. no client found for the subscriber")
			completeSuccess = false
			continue
		}

		if _, _, err := client.Send(context.Background(), *cloudEvent); err != nil {
			logger.WithError(err).WithField("target", subscriber).Warnln("failed to send the event")
			completeSuccess = false
			continue
		}
	}

	response := "dispatched event"
	if !completeSuccess {
		response = fmt.Sprintf("%s.%s", response, " although some of the dispatch operations failed, check logs for more info")
	}

	logger.Infoln(response)
	return nil
}

// transformEvent transforms an event from gateway server into a CloudEvent
// See https://github.com/cloudevents/spec for more info.
func (gatewayContext *GatewayContext) transformEvent(gatewayEvent *gateways.Event) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent(cloudevents.VersionV03)
	event.SetID(fmt.Sprintf("%x", uuid.New()))
	event.SetSpecVersion(cloudevents.VersionV03)
	event.SetType(string(gatewayContext.gateway.Spec.Type))
	event.SetSource(gatewayContext.gateway.Name)
	event.SetDataContentType("application/json")
	event.SetSubject(gatewayEvent.Name)
	event.SetTime(time.Now())
	if err := event.SetData(gatewayEvent.Payload); err != nil {
		return nil, err
	}
	return &event, nil
}
