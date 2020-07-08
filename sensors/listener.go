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

	"github.com/Knetic/govaluate"
	"github.com/antonmedv/expr"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensordependencies "github.com/argoproj/argo-events/sensors/dependencies"
	sensortriggers "github.com/argoproj/argo-events/sensors/triggers"
)

// ListenEvents watches and handles events received from the gateway.
func (sensorCtx *SensorContext) ListenEvents() error {
	errCh := make(chan error)
	ctx := context.Background()
	cctx, cancel := context.WithCancel(ctx)
	sensorCtx.listenEventsOverEventBus(cctx, errCh)
	err := <-errCh
	cancel()
	sensorCtx.Logger.WithError(err).Errorln("subscription failure. stopping sensor operations")
	return nil
}

func (sensorCtx *SensorContext) listenEventsOverEventBus(ctx context.Context, errCh chan<- error) {
	sensor := sensorCtx.Sensor
	// Get a mapping of dependencyExpression: []triggers
	triggerMapping := make(map[string][]v1alpha1.Trigger)
	for _, trigger := range sensor.Spec.Triggers {
		depExpr, err := sensorCtx.getDependencyExpression(trigger)
		if err != nil {
			errCh <- err
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
				errCh <- err
			}
			depNames := unique(expr.Vars())
			deps := []eventbusdriver.Dependency{}
			for _, depName := range depNames {
				dep, ok := depMapping[depName]
				if !ok {
					errCh <- errors.Errorf("Dependency expression and dependency list do not match, %s is not found", depName)
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
			ebDriver, err := eventbus.GetDriver(*sensorCtx.EventBusConfig, sensorCtx.EventBusSubject, clientID, sensorCtx.Logger)
			if err != nil {
				sensorCtx.Logger.WithError(err).Errorln("Failed to get event bus driver")
				errCh <- err
			}
			triggerNames := []string{}
			for _, t := range triggers {
				triggerNames = append(triggerNames, t.Template.Name)
			}
			conn, err := ebDriver.Connect()
			if err != nil {
				sensorCtx.Logger.WithError(err).Errorln("Failed to connect to event bus")
				errCh <- err
			}
			defer conn.Close()
			sensorCtx.Logger.Infof("Started to subscribe events for triggers %s with client %s", fmt.Sprintf("[%s]", strings.Join(triggerNames, " ")), clientID)
			err = ebDriver.SubscribeEventSources(ctx, conn, depExpression, deps, func(depName string, event cloudevents.Event) bool {
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
					sensorCtx.Logger.WithError(err).Errorln("Failed to apply filters")
					return false
				}
				return result
			}, func(events map[string]cloudevents.Event) {
				err := sensorCtx.triggerActions(events, triggers)
				if err != nil {
					sensorCtx.Logger.WithError(err).Errorln("Failed to trigger actions")
				}
			})
			if err != nil {
				sensorCtx.Logger.WithError(err).Errorln("Failed to subscribe to event bus")
				errCh <- err
			}
		}(ctx, k, v)
	}
}

func (sensorCtx *SensorContext) triggerActions(events map[string]cloudevents.Event, triggers []v1alpha1.Trigger) error {
	log := sensorCtx.Logger
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
		triggerImpl := sensorCtx.GetTrigger(&trigger)
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

func (sensorCtx *SensorContext) getDependencyExpression(trigger v1alpha1.Trigger) (string, error) {
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
		program, err := expr.Compile(groupDepExpr, expr.Env(depGroupMapping))
		if err != nil {
			sensorCtx.Logger.WithError(err).Errorln("Failed to compile group dependency expression")
			return "", err
		}
		result, err := expr.Run(program, depGroupMapping)
		if err != nil {
			sensorCtx.Logger.WithError(err).Errorln("Failed to parse group dependency expression")
			return "", err
		}
		depExpression = fmt.Sprintf("%v", result)
	} else {
		deps := []string{}
		for _, dep := range sensor.Spec.Dependencies {
			deps = append(deps, dep.Name)
		}
		depExpression = strings.Join(deps, "&&")
	}
	sensorCtx.Logger.Infof("Dependency expression for trigger %s before simlification: %s", trigger.Template.Name, depExpression)
	boolSimplifier, err := common.NewBoolExpression(depExpression)
	if err != nil {
		sensorCtx.Logger.WithError(err).Errorln("Invalid dependency expression")
		return "", err
	}
	result := boolSimplifier.GetExpression()
	sensorCtx.Logger.Infof("Dependency expression for trigger %s after simlification: %s", trigger.Template.Name, result)
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
