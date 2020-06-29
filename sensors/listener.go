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
	"encoding/json"
	"fmt"
	"strings"
	"io/ioutil"
	"net/http"
	"hash/fnv"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/go-nats"
	"github.com/antonmedv/expr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/dependencies"
	"github.com/argoproj/argo-events/sensors/types"
)

// ListenEvents watches and handles events received from the gateway.
func (sensorCtx *SensorContext) ListenEvents() error {
	// start processing the update Notification NotificationQueue
	go func() {
		for e := range sensorCtx.NotificationQueue {
			sensorCtx.processQueue(e)
		}
	}()

	errCh := make(chan error)
	if sensorCtx.EventBusConfig != nil {
		go func() {
			if err := sensorCtx.listenEventsOverEventBus(); err != nil {
				errCh <- errors.Wrap(err, "failed to listen events over EventBus")
			}
		}()
	} else {
		// TODO: unsupport these.
		sensorCtx.Logger.Warn("spec.subscription is deprecated, will be unsupported soon, please use EventBus instead")
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
	}
	err := <-errCh
	sensorCtx.Logger.WithError(err).Errorln("subscription failure. stopping sensor operations")

	return nil
}

func (sensorCtx *SensorContext) listenEventsOverEventBus() error {
	sensor := sensorCtx.Sensor
	// Get a mapping of dependencyExpression: []triggers
	triggerMapping := make(map[string][]v1alpha1.Trigger)
	for _, trigger := range sensor.Spec.Triggers {
		depExpr, err := sensorCtx.getDependencyExpression(trigger)
		if err != nil {
			return nil, err
		}
		triggers, ok := triggerMapping[depExpr]
		if !ok {
			triggers = []v1alpha1.Trigger{}
		}
		triggers = append(triggers, trigger)
		triggerMapping[depExpr] = triggers
	}

	deps := []eventbusdriver.Dependency{}
	for _, d := range sensor.Spec.Dependencies {
		esName := d.EventSourceName
		if esName == "" {
			esName = d.GatewayName
		}
		ebd := eventbusdriver.Dependency{
			Name: d.Name,
			EventSourceName: esName,
			EventName: d.EventName,
		}
		deps = append(deps, ebd)
	}

	for k, v := range triggerMapping{
		// Generate clientID with hash code
		h := fnv.New32a()
		h.Write([]byte(k))
		clientID := fmt.Sprintf("client-%v", h.Sum32())
		ebDriver, err := eventbus.GetDriver(*sensorCtx.EventBusConfig, sensorCtx.EventBusSubject, clientID, sensorCtx.Logger)
		if err != nil {
			sensorCtx.Logger.WithError(err).Errorln("Failed to get event bus driver")
			return err
		}
		err = ebDriver.Connect()
		if err != nil {
			sensorCtx.Logger.WithError(err).Errorln("Failed to connect to event bus")
			return err
		}
		ebDriver.SubscribeEventSources(k, deps, )
	}
	
	return
}


func (sensorCtx *SensorContext) getDependencyExpression(trigger v1alpha1.Trigger) (string, error) {
	sensor := sensorCtx.Sensor
	var depExpression string
	if len(sensor.Spec.DependencyGroups) > 0 && sensor.Spec.Circuit != "" && trigger.Template.Switch != nil {
		temp := ""
		sw := trigger.Template.Switch
		switch{
		case len(sw.All) > 0:
			temp = strings.Join(sw.All, "&&")
		case len(sw.Any) > 0 :
			temp = strings.Join(sw.Any, "||")
		default:
			return "", errors.New("invalid trigger switch")
		}
		groupDepExpr := fmt.Sprintf("(%s) && (%s)", sensor.Spec.Circuit, temp)
		depGroupMapping := make(map[string]string)
		for _, depGroup := range sensor.Spec.DependencyGroups {
			depGroupMapping[depGroup.Name] = fmt.Sprintf("(%s)", strings.Join(depGroup.Dependencies, "&&"))
		}
		groupDepExpr = strings.ReplaceAll(groupDepExpr, "&&", " + \"&&\" + ")
		groupDepExpr = strings.ReplaceAll(groupDepExpr, "||", " + \"||\" + ")
		program, err := expr.Compile(groupDepExpr, expr.Env(depGroupMapping))
		if err != nil {
			sensorCtx.Logger.WithError(err).Errorln("Failed to compile group dependency expression")
			return "", err
		}
		depExpression, err = expr.Run(program, depGroupMapping)
		if err != nil {
			sensorCtx.Logger.WithError(err).Errorln("Failed to parse group dependency expression")
			return "", err
		}
	} else {
		deps := []string
		for _, dep := range sensor.Spec.Dependencies {
			deps = deps.append(deps, dep.Name)
		}
		depExpression = strings.Join(deps, "&&")
	}
	boolSimplifier, err := common.NewBoolExpression(depExpression)
	if err != nil {
		sensorCtx.Logger.WithError(err).Errorln("Invalid dependency expression")
		return "", err
	}
	return boolSimplifier.GetExpression()
}

// // listenEventsOverHTTP listens to events over HTTP
// func (sensorCtx *SensorContext) listenEventsOverHTTP() error {
// 	port := sensorCtx.Sensor.Spec.Subscription.HTTP.Port
// 	if port == 0 {
// 		port = common.SensorServerPort
// 	}

// 	sensorCtx.Logger.WithFields(logrus.Fields{
// 		"port":     port,
// 		"endpoint": "/",
// 	}).Infoln("starting HTTP events receiver")

// 	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
// 		eventBody, err := ioutil.ReadAll(request.Body)
// 		if err != nil {
// 			writer.WriteHeader(http.StatusBadRequest)
// 			_, _ = writer.Write([]byte("failed to parse the event"))
// 			return
// 		}
// 		if err := sensorCtx.handleEvent(eventBody); err != nil {
// 			writer.WriteHeader(http.StatusInternalServerError)
// 			_, _ = writer.Write([]byte("failed to handle the event"))
// 			return
// 		}
// 	})

// 	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
// 		return err
// 	}

// 	return nil
// }

// listenEventsOverNATS listens to events over NATS
// func (sensorCtx *SensorContext) listenEventsOverNATS() error {
// 	subscription := sensorCtx.Sensor.Spec.Subscription.NATS

// 	conn, err := nats.Connect(subscription.ServerURL)
// 	if err != nil {
// 		return err
// 	}

// 	logger := sensorCtx.Logger.WithFields(logrus.Fields{
// 		"url":     subscription.ServerURL,
// 		"subject": subscription.Subject,
// 	})

// 	logger.Infoln("starting NATS events subscriber")

// 	_, err = conn.Subscribe(subscription.Subject, func(msg *nats.Msg) {
// 		if err := sensorCtx.handleEvent(msg.Data); err != nil {
// 			logger.WithError(err).Errorln("failed to process the event")
// 		}
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

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
