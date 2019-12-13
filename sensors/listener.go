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
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/dependencies"
	cloudevents "github.com/cloudevents/sdk-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WatchEventsFromGateways watches and handles events received from the gateway.
func (sensorCtx *SensorContext) ListenEvents() error {
	// start processing the update Notification NotificationQueue
	go func() {
		for e := range sensorCtx.NotificationQueue {
			sensorCtx.processQueue(e)
		}
	}()

	// sync Sensor resource after updates
	go sensorCtx.syncSensor(context.Background())

	port := common.SensorServerPort
	if sensorCtx.Sensor.Spec.Port != nil {
		port = *sensorCtx.Sensor.Spec.Port
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

func cloudEventConverter(event *cloudevents.Event) (*apicommon.Event, error) {
	data, err := event.DataBytes()
	if err != nil {
		return nil, err
	}
	return &apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: event.DataContentType(),
			Source:          event.Source(),
			SpecVersion:     event.SpecVersion(),
			Type:            event.Type(),
			Time:            metav1.MicroTime{Time: event.Time()},
			ID:              event.ID(),
			Subject:         event.Subject(),
		},
		Data: data,
	}, nil
}

// handleEvent handles a cloudevent, validates and sends it over internal Event NotificationQueue
func (sensorCtx *SensorContext) handleEvent(ctx context.Context, event *cloudevents.Event) bool {
	internalEvent, err := cloudEventConverter(event)
	if err != nil {
		sensorCtx.Logger.WithError(err).Errorln("failed to parse the cloud event payload")
		return false
	}
	// Resolve Dependency
	// validate whether the Event is from gateway that this Sensor is watching
	if eventDependency := dependencies.ResolveDependency(sensorCtx.Sensor, internalEvent); eventDependency != nil {
		// send Event on internal NotificationQueue
		sensorCtx.NotificationQueue <- &Notification{
			Event:            internalEvent,
			EventDependency:  eventDependency,
			NotificationType: v1alpha1.EventNotification,
		}
		return true
	}
	return false
}
