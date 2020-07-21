/*
Copyright 2020 BlackRock, Inc.

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
	"fmt"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/antonmedv/expr"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensordependencies "github.com/argoproj/argo-events/sensors/dependencies"
	sensortriggers "github.com/argoproj/argo-events/sensors/triggers"
)

// ListenEvents watches and handles events received from the gateway.
func (sensorCtx *SensorContext) ListenEvents(ctx context.Context, stopCh <-chan struct{}) error {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	logger := logging.FromContext(cctx)
	sensor := sensorCtx.Sensor
	// Get a mapping of dependencyExpression: []triggers
	triggerMapping := make(map[string][]v1alpha1.Trigger)
	for _, trigger := range sensor.Spec.Triggers {
		depExpr, err := sensorCtx.getDependencyExpression(cctx, trigger)
		if err != nil {
			logger.WithError(err).Errorln("failed to get dependency expression")
			return err
		}
		triggers, ok := triggerMapping[depExpr]
		if !ok {
			triggers = []v1alpha1.Trigger{}
		}
		triggers = append(triggers, trigger)
		triggerMapping[depExpr] = triggers
	}

	depMapping := make(map[string]v1alpha1.EventDependency)
	for _, d := range sensor.Spec.Dependencies {
		depMapping[d.Name] = d
	}

	for k, v := range triggerMapping {
		go func(ctx context.Context, depExpression string, triggers []v1alpha1.Trigger) {
			// Calculate dependencies of each group of triggers.
			de := strings.ReplaceAll(depExpression, "-", "\\-")
			expr, err := govaluate.NewEvaluableExpression(de)
			if err != nil {
				logger.WithError(err).Errorln("failed to get new evaluable expression")
				return
			}
			depNames := unique(expr.Vars())
			deps := []eventbusdriver.Dependency{}
			for _, depName := range depNames {
				dep, ok := depMapping[depName]
				if !ok {
					logger.Errorf("Dependency expression and dependency list do not match, %s is not found", depName)
					return
				}
				esName := dep.EventSourceName
				if esName == "" {
					esName = dep.GatewayName
				}
				d := eventbusdriver.Dependency{
					Name:            dep.Name,
					EventSourceName: esName,
					EventName:       dep.EventName,
				}
				deps = append(deps, d)
			}

			// Generate clientID with hash code
			hashKey := fmt.Sprintf("%s-%s", sensorCtx.Sensor.Name, depExpression)
			clientID := fmt.Sprintf("client-%v", common.Hasher(hashKey))
			ebDriver, err := eventbus.GetDriver(ctx, *sensorCtx.EventBusConfig, sensorCtx.EventBusSubject, clientID)
			if err != nil {
				logger.WithError(err).Errorln("Failed to get event bus driver")
				return
			}
			triggerNames := []string{}
			for _, t := range triggers {
				triggerNames = append(triggerNames, t.Template.Name)
			}
			conn, err := ebDriver.Connect()
			if err != nil {
				logger.WithError(err).Errorln("Failed to connect to event bus")
				return
			}
			defer conn.Close()

			filterFunc := func(depName string, event cloudevents.Event) bool {
				dep, ok := depMapping[depName]
				if !ok {
					return false
				}
				if dep.Filters == nil {
					return true
				}
				e := convertEvent(event)
				result, err := sensordependencies.Filter(e, dep.Filters)
				if err != nil {
					logger.WithError(err).Errorln("Failed to apply filters")
					return false
				}
				return result
			}

			actionFunc := func(events map[string]cloudevents.Event) {
				err := sensorCtx.triggerActions(ctx, events, triggers)
				if err != nil {
					logger.WithError(err).Errorln("Failed to trigger actions")
				}
			}

			// Attempt to reconnect
			closeSubCh := make(chan struct{})
			go func(ctx context.Context, dvr eventbusdriver.Driver) {
				logger.Infof("starting eventbus connection daemon for client %s...", clientID)
				for {
					select {
					case <-ctx.Done():
						logger.Infof("exiting eventbus connection daemon for client %s...", clientID)
						return
					default:
						time.Sleep(5 * time.Second)
					}
					if conn == nil || conn.IsClosed() {
						logger.Info("NATS connection lost, reconnecting...")
						conn, err = dvr.Connect()
						if err != nil {
							logger.WithError(err).Errorf("failed to reconnect to eventbus, client: %s", clientID)
							continue
						}
						logger.Infof("reconnected the NATS streaming server for client %s...", clientID)
						closeSubCh <- struct{}{}
						time.Sleep(2 * time.Second)
						err = ebDriver.SubscribeEventSources(ctx, conn, closeSubCh, depExpression, deps, filterFunc, actionFunc)
					}
				}
			}(ctx, ebDriver)

			logger.Infof("Started to subscribe events for triggers %s with client %s", fmt.Sprintf("[%s]", strings.Join(triggerNames, " ")), clientID)
			err = ebDriver.SubscribeEventSources(ctx, conn, closeSubCh, depExpression, deps, filterFunc, actionFunc)
			if err != nil {
				logger.WithError(err).Errorln("Failed to subscribe to event bus")
				return
			}
		}(cctx, k, v)
	}
	logger.Info("Sensor started.")
	<-stopCh
	logger.Info("Shutting down...")
	return nil
}

func (sensorCtx *SensorContext) triggerActions(ctx context.Context, events map[string]cloudevents.Event, triggers []v1alpha1.Trigger) error {
	log := logging.FromContext(ctx)
	eventsMapping := make(map[string]*v1alpha1.Event)
	for k, v := range events {
		eventsMapping[k] = convertEvent(v)
	}
	for _, trigger := range triggers {
		if err := sensortriggers.ApplyTemplateParameters(eventsMapping, &trigger); err != nil {
			log.Errorf("failed to apply template parameters, %v", err)
			return err
		}

		log.WithField("triggerName", trigger.Template.Name).Infoln("resolving the trigger implementation")
		triggerImpl := sensorCtx.GetTrigger(ctx, &trigger)
		if triggerImpl == nil {
			log.WithField("triggerName", trigger.Template.Name).Errorln("failed to get the specific trigger implementation. continuing to next trigger if any")
			continue
		}

		log.WithField("triggerName", trigger.Template.Name).Infoln("fetching trigger resource if any")
		obj, err := triggerImpl.FetchResource()
		if err != nil {
			return err
		}
		if obj == nil {
			log.WithField("triggerName", trigger.Template.Name).Warnln("trigger resource is empty")
			continue
		}

		log.WithField("triggerName", trigger.Template.Name).Infoln("applying resource parameters if any")
		updatedObj, err := triggerImpl.ApplyResourceParameters(eventsMapping, obj)
		if err != nil {
			return err
		}

		log.WithField("triggerName", trigger.Template.Name).Infoln("executing the trigger resource")
		newObj, err := triggerImpl.Execute(eventsMapping, updatedObj)
		if err != nil {
			return err
		}
		log.WithField("triggerName", trigger.Template.Name).Infoln("trigger resource successfully executed")

		log.WithField("triggerName", trigger.Template.Name).Infoln("applying trigger policy")
		if err := triggerImpl.ApplyPolicy(newObj); err != nil {
			return err
		}

		log.WithField("triggerName", trigger.Template.Name).Infoln("successfully processed the trigger")
	}
	return nil
}

func (sensorCtx *SensorContext) getDependencyExpression(ctx context.Context, trigger v1alpha1.Trigger) (string, error) {
	logger := logging.FromContext(ctx)
	sensor := sensorCtx.Sensor
	var depExpression string
	if len(sensor.Spec.DependencyGroups) > 0 && sensor.Spec.Circuit != "" && trigger.Template.Switch != nil {
		temp := ""
		sw := trigger.Template.Switch
		switch {
		case len(sw.All) > 0:
			temp = strings.Join(sw.All, "&&")
		case len(sw.Any) > 0:
			temp = strings.Join(sw.Any, "||")
		default:
			return "", errors.New("invalid trigger switch")
		}
		groupDepExpr := fmt.Sprintf("(%s) && (%s)", sensor.Spec.Circuit, temp)
		depGroupMapping := make(map[string]string)
		for _, depGroup := range sensor.Spec.DependencyGroups {
			key := strings.ReplaceAll(depGroup.Name, "-", "_")
			depGroupMapping[key] = fmt.Sprintf("(%s)", strings.Join(depGroup.Dependencies, "&&"))
		}
		groupDepExpr = strings.ReplaceAll(groupDepExpr, "&&", " + \"&&\" + ")
		groupDepExpr = strings.ReplaceAll(groupDepExpr, "||", " + \"||\" + ")
		groupDepExpr = strings.ReplaceAll(groupDepExpr, "-", "_")
		groupDepExpr = strings.ReplaceAll(groupDepExpr, "(", "\"(\"+")
		groupDepExpr = strings.ReplaceAll(groupDepExpr, ")", "+\")\"")
		program, err := expr.Compile(groupDepExpr, expr.Env(depGroupMapping))
		if err != nil {
			logger.WithError(err).Errorln("Failed to compile group dependency expression")
			return "", err
		}
		result, err := expr.Run(program, depGroupMapping)
		if err != nil {
			logger.WithError(err).Errorln("Failed to parse group dependency expression")
			return "", err
		}
		depExpression = fmt.Sprintf("%v", result)
		depExpression = strings.ReplaceAll(depExpression, "\"(\"", "(")
		depExpression = strings.ReplaceAll(depExpression, "\")\"", ")")
	} else {
		deps := []string{}
		for _, dep := range sensor.Spec.Dependencies {
			deps = append(deps, dep.Name)
		}
		depExpression = strings.Join(deps, "&&")
	}
	logger.Infof("Dependency expression for trigger %s before simlification: %s", trigger.Template.Name, depExpression)
	boolSimplifier, err := common.NewBoolExpression(depExpression)
	if err != nil {
		logger.WithError(err).Errorln("Invalid dependency expression")
		return "", err
	}
	result := boolSimplifier.GetExpression()
	logger.Infof("Dependency expression for trigger %s after simlification: %s", trigger.Template.Name, result)
	return result, nil
}

func convertEvent(event cloudevents.Event) *v1alpha1.Event {
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
		Data: event.Data(),
	}
}

func unique(stringSlice []string) []string {
	if len(stringSlice) == 0 {
		return stringSlice
	}
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
