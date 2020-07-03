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
	"encoding/json"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/eventbus"
	"github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/gateways"
)

var (
	ebDriver driver.Driver
	conn     driver.Connection
)

// updateSubscriberClients updates the active clients for event subscribers
func (gatewayContext *GatewayContext) updateSubscriberClients() {
	if ebDriver == nil {
		d, err := eventbus.GetDriver(*gatewayContext.eventBusConfig, gatewayContext.eventBusSubject, gatewayContext.podName, gatewayContext.logger)
		if err != nil {
			gatewayContext.logger.WithError(err).Warnln("failed to get eventbus driver")
			return
		}
		ebDriver = d
	}
	if conn == nil || conn.IsClosed() {
		c, err := ebDriver.Connect()
		if err != nil {
			gatewayContext.logger.WithError(err).Warnln("failed to connect to eventbus")
			return
		}
		conn = c
	}
}

// dispatchEvent dispatches event to gateway transformer for further processing
func (gatewayContext *GatewayContext) dispatchEvent(gatewayEvent *gateways.Event) error {
	logger := gatewayContext.logger.WithField(common.LabelEventSource, gatewayEvent.Name)
	logger.Infoln("dispatching event to eventbus")

	cloudEvent, err := gatewayContext.transformEvent(gatewayEvent)
	if err != nil {
		logger.WithError(err).Errorln("failed to convert to cloudevent")
		return err
	}

	eventBody, err := json.Marshal(cloudEvent)
	if err != nil {
		logger.WithError(err).Errorln("failed to marshal the event")
		return err
	}

	gatewayContext.updateSubscriberClients()

	err = ebDriver.Publish(conn, eventBody)
	if err != nil {
		logger.WithError(err).Infoln("Failed to send a message to eventbus")
		return err
	}
	logger.WithFields(logrus.Fields{
		"eventName": gatewayEvent.Name,
	}).Infoln("successfully sent a message to the eventbus")
	return nil
}

// transformEvent transforms an event from gateway server into a CloudEvent
// See https://github.com/cloudevents/spec for more info.
func (gatewayContext *GatewayContext) transformEvent(gatewayEvent *gateways.Event) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent()
	event.SetID(fmt.Sprintf("%x", uuid.New()))
	event.SetType(string(gatewayContext.gateway.Spec.Type))
	event.SetSource(gatewayContext.gateway.Spec.EventSourceRef.Name)
	event.SetSubject(gatewayEvent.Name)
	event.SetTime(time.Now())
	err := event.SetData(cloudevents.ApplicationJSON, gatewayEvent.Payload)
	if err != nil {
		return nil, err
	}
	return &event, nil
}
