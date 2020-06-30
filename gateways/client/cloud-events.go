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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
)

// updateSubscriberClients updates the active clients for event subscribers
func (gatewayContext *GatewayContext) updateSubscriberClients() {
	if gatewayContext.gateway.Spec.Subscribers == nil {
		return
	}

	if gatewayContext.natsSubscribers == nil {
		gatewayContext.natsSubscribers = make(map[string]*nats.Conn)
	}

	// nats subscribers
	for _, subscriber := range gatewayContext.gateway.Spec.Subscribers.NATS {
		if _, ok := gatewayContext.natsSubscribers[subscriber.Name]; !ok {
			conn, err := nats.Connect(subscriber.ServerURL)
			if err != nil {
				gatewayContext.logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to connect to subscriber")
				continue
			}

			gatewayContext.logger.WithField("subscriber", subscriber).Infoln("added a client for the subscriber")
			gatewayContext.natsSubscribers[subscriber.Name] = conn
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

	cloudEvent := gatewayContext.transformEvent(gatewayEvent)

	completeSuccess := true

	eventBody, err := json.Marshal(cloudEvent)
	if err != nil {
		logger.WithError(err).Errorln("failed to marshal the event")
		return err
	}

	// http subscribers
	for _, subscriber := range gatewayContext.gateway.Spec.Subscribers.HTTP {
		request, err := http.NewRequest(http.MethodPost, subscriber, bytes.NewReader(eventBody))
		if err != nil {
			logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to construct http request for the event")
			completeSuccess = false
			continue
		}

		response, err := gatewayContext.httpClient.Do(request)
		if err != nil {
			logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to send http request for the event")
			completeSuccess = false
			continue
		}

		logger.WithFields(logrus.Fields{
			"status":     response.Status,
			"subscriber": subscriber,
		}).Infoln("successfully sent event to the subscriber")
	}

	// NATS subscribers
	for _, subscriber := range gatewayContext.gateway.Spec.Subscribers.NATS {
		conn, ok := gatewayContext.natsSubscribers[subscriber.Name]
		if !ok {
			gatewayContext.logger.WithField("subscriber", subscriber).Warnln("unable to send event. no client found for the subscriber")
			completeSuccess = false
			continue
		}

		if err := conn.Publish(subscriber.Subject, eventBody); err != nil {
			logger.WithError(err).WithField("subscriber", subscriber).Warnln("failed to publish the event")
			completeSuccess = false
			continue
		}

		logger.WithFields(logrus.Fields{
			"subscriber": subscriber.Name,
			"subject":    subscriber.Subject,
		}).Infoln("successfully published event on the subject")
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
func (gatewayContext *GatewayContext) transformEvent(gatewayEvent *gateways.Event) *cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetID(fmt.Sprintf("%x", uuid.New()))
	event.SetType(string(gatewayContext.gateway.Spec.Type))
	event.SetSource(gatewayContext.gateway.Spec.EventSourceRef.Name)
	event.SetSubject(gatewayEvent.Name)
	event.SetTime(time.Now())
	event.SetData(cloudevents.ApplicationJSON, gatewayEvent.Payload)
	return &event
}
