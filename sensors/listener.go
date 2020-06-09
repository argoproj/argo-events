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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/dependencies"
	"github.com/argoproj/argo-events/sensors/types"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/nats-io/go-nats"
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

	// listen events over http
	if sensorCtx.Sensor.Spec.Subscription.HTTP != nil {
		go func() {
			if err := sensorCtx.listenEventsOverHTTP(); err != nil {
				errCh <- errors.Wrap(err, "failed to listen events over HTTP subscription")
			}
		}()
	}

	// listen events over nats
	if sensorCtx.Sensor.Spec.Subscription.NATS != nil {
		go func() {
			if err := sensorCtx.listenEventsOverNATS(); err != nil {
				errCh <- errors.Wrap(err, "failed to listen events over NATS subscription")
			}
		}()
	}

	err := <-errCh
	sensorCtx.Logger.WithError(err).Errorln("subscription failure. stopping sensor operations")

	return nil
}

// listenEventsOverHTTP listens to events over HTTP
func (sensorCtx *SensorContext) listenEventsOverHTTP() error {
	port := sensorCtx.Sensor.Spec.Subscription.HTTP.Port
	if port == 0 {
		port = common.SensorServerPort
	}

	sensorCtx.Logger.WithFields(logrus.Fields{
		"port":     port,
		"endpoint": "/",
	}).Infoln("starting HTTP events receiver")

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		eventBody, err := ioutil.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			writer.Write([]byte("failed to parse the event"))
			return
		}
		if err := sensorCtx.handleEvent(eventBody); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			writer.Write([]byte("failed to handle the event"))
			return
		}
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		return err
	}

	return nil
}

// listenEventsOverNATS listens to events over NATS
func (sensorCtx *SensorContext) listenEventsOverNATS() error {
	subscription := sensorCtx.Sensor.Spec.Subscription.NATS

	conn, err := nats.Connect(subscription.ServerURL)
	if err != nil {
		return err
	}

	logger := sensorCtx.Logger.WithFields(logrus.Fields{
		"url":     subscription.ServerURL,
		"subject": subscription.Subject,
	})

	logger.Infoln("starting NATS events subscriber")

	_, err = conn.Subscribe(subscription.Subject, func(msg *nats.Msg) {
		if err := sensorCtx.handleEvent(msg.Data); err != nil {
			logger.WithError(err).Errorln("failed to process the event")
		}
	})
	if err != nil {
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
func (sensorCtx *SensorContext) handleEvent(eventBody []byte) error {
	var event *cloudevents.Event
	if err := json.Unmarshal(eventBody, &event); err != nil {
		return err
	}

	internalEvent, err := cloudEventConverter(event)
	if err != nil {
		return errors.Wrap(err, "failed to parse the cloudevent")
	}

	sensorCtx.Logger.WithFields(logrus.Fields{
		"source":  event.Context.GetSource(),
		"subject": event.Context.GetSubject(),
	}).Infoln("received event")

	// Resolve Dependency
	// validate whether the event is from gateway that this sensor is watching
	if eventDependency := dependencies.ResolveDependency(sensorCtx.Sensor.Spec.Dependencies, internalEvent, sensorCtx.Logger); eventDependency != nil {
		sensorCtx.NotificationQueue <- &types.Notification{
			Event:            internalEvent,
			EventDependency:  eventDependency,
			NotificationType: v1alpha1.EventNotification,
		}
	}
	return nil
}
