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

package sensors

import (
	"context"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go"
)

// WatchEventsFromGateways watches and handles events received from the gateway.
func (sensorCtx *sensorContext) ListenEvents() error {
	// start processing the update notification notificationQueue
	go func() {
		for e := range sensorCtx.notificationQueue {
			sensorCtx.processNotificationQueue(e)
		}
	}()

	// sync sensor resource after updates
	go sensorCtx.syncSensor(context.Background())

	port := common.SensorServerPort
	if sensorCtx.sensor.Spec.Port != nil {
		port = *sensorCtx.sensor.Spec.Port
	}

	t, err := cloudevents.NewHTTPTransport(
		cloudevents.WithPort(port),
		cloudevents.WithPath("/"),
	)
	if err != nil {
		return err
	}

	client, err := cloudevents.NewClient(t)
	if err != nil {
		return err
	}

	if err := client.StartReceiver(context.Background(), sensorCtx.handleEvent); err != nil {
		return err
	}

	return nil
}

// handleEvent handles a cloudevent, validates and sends it over internal event notificationQueue
func (sensorCtx *sensorContext) handleEvent(ctx context.Context, event *cloudevents.Event) bool {
	// validate whether the event is from gateway that this sensor is watching
	if eventDependency := sensorCtx.resolveDependency(event); eventDependency != nil {
		// send event on internal notificationQueue
		sensorCtx.notificationQueue <- &notification{
			event:            event,
			eventDependency:  eventDependency,
			notificationType: v1alpha1.EventNotification,
		}
		return true
	}
	return false
}
