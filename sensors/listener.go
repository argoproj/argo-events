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
	"github.com/argoproj/argo-events/sensors/dependencies"
	"github.com/argoproj/argo-events/sensors/types"
	cloudevents "github.com/cloudevents/sdk-go"
	cloudeventsnats "github.com/cloudevents/sdk-go/pkg/cloudevents/transport/nats"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListenEvents watches and handles events received from the gateway.
func (sensorCtx *SensorContext) ListenEvents() error {
	// start processing the update Notification NotificationQueue
	go func() {
		for e := range sensorCtx.NotificationQueue {
			sensorCtx.processQueue(e)
		}
	}()

	// sync Sensor resource after updates
	go sensorCtx.syncSensor(context.Background())

	errCh := make(chan error)
	httpCtx, httpCancel := context.WithCancel(context.Background())
	natsCtx, natsCancel := context.WithCancel(context.Background())

	// listen events over http
	if sensorCtx.Sensor.Spec.Subscription.HTTP != nil {
		go func() {
			if err := sensorCtx.listenEventsOverHTTP(httpCtx); err != nil {
				errCh <- errors.Wrap(err, "failed to listen events over HTTP subscription")
			}
		}()
	}

	// listen events over nats
	if sensorCtx.Sensor.Spec.Subscription.NATS != nil {
		go func() {
			if err := sensorCtx.listenEventsOverNATS(natsCtx); err != nil {
				errCh <- errors.Wrap(err, "failed to listen events over NATS subscription")
			}
		}()
	}

	err := <-errCh
	sensorCtx.Logger.WithError(err).Errorln("subscription failure. stopping sensor operations")

	httpCancel()
	natsCancel()

	return nil
}

// listenEventsOverHTTP listens to events over HTTP
func (sensorCtx *SensorContext) listenEventsOverHTTP(ctx context.Context) error {
	port := sensorCtx.Sensor.Spec.Subscription.HTTP.Port
	if port == 0 {
		port = common.SensorServerPort
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

	sensorCtx.Logger.WithFields(logrus.Fields{
		"port":     port,
		"endpoint": "/",
	}).Infoln("starting HTTP cloudevents receiver")

	if err := client.StartReceiver(ctx, sensorCtx.handleEvent); err != nil {
		return err
	}

	return nil
}

// listenEventsOverNATS listens to events over NATS
func (sensorCtx *SensorContext) listenEventsOverNATS(ctx context.Context) error {
	subscription := sensorCtx.Sensor.Spec.Subscription.NATS
	t, err := cloudeventsnats.New(subscription.ServerURL, subscription.Subject)
	if err != nil {
		return err
	}

	client, err := cloudevents.NewClient(t)
	if err != nil {
		return err
	}

	sensorCtx.Logger.WithFields(logrus.Fields{
		"url":     subscription.ServerURL,
		"subject": subscription.Subject,
	}).Infoln("starting NATS cloudevents receiver")

	if err := client.StartReceiver(ctx, sensorCtx.handleEvent); err != nil {
		return err
	}

	return nil
}

func cloudEventConverter(event *cloudevents.Event) (*v1alpha1.Event, error) {
	data, err := event.DataBytes()
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			DataContentType: event.Context.GetDataContentType(),
			Source:          event.Context.GetSource(),
			SpecVersion:     event.Context.GetSpecVersion(),
			Type:            event.Context.GetType(),
			Time:            metav1.Time{Time: event.Context.GetTime()},
			ID:              event.Context.GetID(),
			Subject:         event.Context.GetSubject(),
		},
		Data: data,
	}, nil
}

// handleEvent handles a cloudevent, validates and sends it over internal event notification queue
func (sensorCtx *SensorContext) handleEvent(ctx context.Context, event cloudevents.Event) error {
	internalEvent, err := cloudEventConverter(&event)
	if err != nil {
		return errors.Wrap(err, "failed to parse the cloudevent")
	}

	sensorCtx.Logger.WithFields(logrus.Fields{
		"source":  event.Context.GetSource(),
		"subject": event.Context.GetSubject(),
	}).Infoln("received event")

	// Resolve Dependency
	// validate whether the event is from gateway that this sensor is watching
	if eventDependency := dependencies.ResolveDependency(sensorCtx.Sensor.Spec.Dependencies, internalEvent); eventDependency != nil {
		sensorCtx.NotificationQueue <- &types.Notification{
			Event:            internalEvent,
			EventDependency:  eventDependency,
			NotificationType: v1alpha1.EventNotification,
		}
	}
	return nil
}
