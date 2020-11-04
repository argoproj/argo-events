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
	"sync"
	"sync/atomic"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/antonmedv/expr"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensordependencies "github.com/argoproj/argo-events/sensors/dependencies"
	sensortriggers "github.com/argoproj/argo-events/sensors/triggers"
)

func subscribeOnce(subLock *uint32, subscribe func()) {
	// acquire subLock if not already held
	if !atomic.CompareAndSwapUint32(subLock, 0, 1) {
		return
	}

	subscribe()
}

// ListenEvents watches and handles events received from the gateway.
func (sensorCtx *SensorContext) ListenEvents(ctx context.Context, stopCh <-chan struct{}) error {
	logger := logging.FromContext(ctx).Desugar()
	sensor := sensorCtx.Sensor
	// Get a mapping of dependencyExpression: []triggers
	triggerMapping := make(map[string][]v1alpha1.Trigger)
	for _, trigger := range sensor.Spec.Triggers {
		depExpr, err := sensorCtx.getDependencyExpression(ctx, trigger)
		if err != nil {
			logger.Error("failed to get dependency expression", zap.Error(err))
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

	cctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	for k, v := range triggerMapping {
		wg.Add(1)
		go func(depExpression string, triggers []v1alpha1.Trigger) {
			defer wg.Done()
			// Calculate dependencies of each group of triggers.
			de := strings.ReplaceAll(depExpression, "-", "\\-")
			expr, err := govaluate.NewEvaluableExpression(de)
			if err != nil {
				logger.Error("failed to get new evaluable expression", zap.Error(err))
				return
			}
			depNames := unique(expr.Vars())
			deps := []eventbusdriver.Dependency{}
			for _, depName := range depNames {
				dep, ok := depMapping[depName]
				if !ok {
					logger.Sugar().Errorf("Dependency expression and dependency list do not match, %s is not found", depName)
					return
				}
				d := eventbusdriver.Dependency{
					Name:            dep.Name,
					EventSourceName: dep.EventSourceName,
					EventName:       dep.EventName,
				}
				deps = append(deps, d)
			}

			// Generate clientID with hash code
			hashKey := fmt.Sprintf("%s-%s", sensorCtx.Sensor.Name, depExpression)
			clientID := fmt.Sprintf("client-%v", common.Hasher(hashKey))
			ebDriver, err := eventbus.GetDriver(cctx, *sensorCtx.EventBusConfig, sensorCtx.EventBusSubject, clientID)
			if err != nil {
				logger.Error("failed to get event bus driver", zap.Error(err))
				return
			}
			triggerNames := []string{}
			for _, t := range triggers {
				triggerNames = append(triggerNames, t.Template.Name)
			}
			var conn eventbusdriver.Connection
			err = common.Connect(&common.DefaultRetry, func() error {
				var err error
				conn, err = ebDriver.Connect()
				return err
			})
			if err != nil {
				logger.Fatal("failed to connect to event bus", zap.Error(err))
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
					logger.Error("failed to apply filters", zap.Error(err))
					return false
				}
				return result
			}

			actionFunc := func(events map[string]cloudevents.Event) {
				err := sensorCtx.triggerActions(cctx, events, triggers)
				if err != nil {
					logger.Error("failed to trigger actions", zap.Error(err))
				}
			}

			var subLock uint32
			wg1 := &sync.WaitGroup{}
			closeSubCh := make(chan struct{})

			subscribeFunc := func() {
				wg1.Add(1)
				go func() {
					defer wg1.Done()
					// release the lock when goroutine exits
					defer atomic.StoreUint32(&subLock, 0)

					logger.Sugar().Infof("started subscribing to events for triggers %s with client %s", fmt.Sprintf("[%s]", strings.Join(triggerNames, " ")), clientID)

					err = ebDriver.SubscribeEventSources(cctx, conn, closeSubCh, depExpression, deps, filterFunc, actionFunc)
					if err != nil {
						logger.Error("failed to subscribe to event bus", zap.Any("clientID", clientID), zap.Error(err))
						return
					}
				}()
			}

			subscribeOnce(&subLock, subscribeFunc)

			logger.Sugar().Infof("starting eventbus connection daemon for client %s...", clientID)
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-cctx.Done():
					logger.Sugar().Infof("exiting eventbus connection daemon for client %s...", clientID)
					ticker.Stop()
					wg1.Wait()
					return
				case <-ticker.C:
					if conn == nil || conn.IsClosed() {
						logger.Info("NATS connection lost, reconnecting...")
						conn, err = ebDriver.Connect()
						if err != nil {
							logger.Error("failed to reconnect to eventbus", zap.Any("clientID", clientID), zap.Error(err))
							continue
						}
						logger.Info("reconnected to NATS streaming server.", zap.Any("clientID", clientID))

						if atomic.LoadUint32(&subLock) == 1 {
							closeSubCh <- struct{}{}
							// give subscription time to close
							time.Sleep(2 * time.Second)
						}
					}

					// create subscription if conn is alive and no subscription is currently held
					if conn != nil && !conn.IsClosed() {
						subscribeOnce(&subLock, subscribeFunc)
					}
				}
			}
		}(k, v)
	}
	logger.Info("Sensor started.")
	<-stopCh
	logger.Info("Shutting down...")
	cancel()
	wg.Wait()
	return nil
}

func (sensorCtx *SensorContext) triggerActions(ctx context.Context, events map[string]cloudevents.Event, triggers []v1alpha1.Trigger) error {
	log := logging.FromContext(ctx)
	eventsMapping := make(map[string]*v1alpha1.Event)
	depNames := make([]string, 0, len(events))
	for k, v := range events {
		eventsMapping[k] = convertEvent(v)
		depNames = append(depNames, k)
	}
	for _, trigger := range triggers {
		if err := sensortriggers.ApplyTemplateParameters(eventsMapping, &trigger); err != nil {
			log.Errorf("failed to apply template parameters, %v", err)
			return err
		}

		log.Debugw("resolving the trigger implementation", "triggerName", trigger.Template.Name)
		triggerImpl := sensorCtx.GetTrigger(ctx, &trigger)
		if triggerImpl == nil {
			log.Errorw("failed to get the specific trigger implementation. continuing to next trigger if any", "triggerName", trigger.Template.Name)
			continue
		}

		log.Debugw("fetching trigger resource if any", "triggerName", trigger.Template.Name)
		obj, err := triggerImpl.FetchResource()
		if err != nil {
			return err
		}
		if obj == nil {
			log.Debugw("trigger resource is empty", "triggerName", trigger.Template.Name)
			continue
		}

		log.Debugw("applying resource parameters if any", "triggerName", trigger.Template.Name)
		updatedObj, err := triggerImpl.ApplyResourceParameters(eventsMapping, obj)
		if err != nil {
			return err
		}

		log.Debugw("executing the trigger resource", "triggerName", trigger.Template.Name)
		newObj, err := triggerImpl.Execute(eventsMapping, updatedObj)
		if err != nil {
			return err
		}
		log.Debugw("trigger resource successfully executed", "triggerName", trigger.Template.Name)

		log.Debugw("applying trigger policy", "triggerName", trigger.Template.Name)
		if err := triggerImpl.ApplyPolicy(newObj); err != nil {
			return err
		}
		log.Infow("successfully processed the trigger", zap.String("triggerName", trigger.Template.Name), zap.Any("triggeredBy", depNames))
	}
	return nil
}

func (sensorCtx *SensorContext) getDependencyExpression(ctx context.Context, trigger v1alpha1.Trigger) (string, error) {
	logger := logging.FromContext(ctx).Desugar()

	// Translate original expression which might contain group names
	// to an expression only contains dependency names
	translate := func(originalExpr string, parameters map[string]string) (string, error) {
		originalExpr = strings.ReplaceAll(originalExpr, "&&", " + \"&&\" + ")
		originalExpr = strings.ReplaceAll(originalExpr, "||", " + \"||\" + ")
		originalExpr = strings.ReplaceAll(originalExpr, "-", "_")
		originalExpr = strings.ReplaceAll(originalExpr, "(", "\"(\"+")
		originalExpr = strings.ReplaceAll(originalExpr, ")", "+\")\"")

		program, err := expr.Compile(originalExpr, expr.Env(parameters))
		if err != nil {
			logger.Error("Failed to compile original dependency expression", zap.Error(err))
			return "", err
		}
		result, err := expr.Run(program, parameters)
		if err != nil {
			logger.Error("Failed to parse original dependency expression", zap.Error(err))
			return "", err
		}
		newExpr := fmt.Sprintf("%v", result)
		newExpr = strings.ReplaceAll(newExpr, "\"(\"", "(")
		newExpr = strings.ReplaceAll(newExpr, "\")\"", ")")
		return newExpr, nil
	}

	sensor := sensorCtx.Sensor
	var depExpression string
	var err error
	switch {
	case trigger.Template.Conditions != "":
		conditions := trigger.Template.Conditions
		// Add all the dependency and dependency group to the parameter mappings
		depGroupMapping := make(map[string]string)
		for _, dep := range sensor.Spec.Dependencies {
			key := strings.ReplaceAll(dep.Name, "-", "_")
			depGroupMapping[key] = dep.Name
		}
		for _, depGroup := range sensor.Spec.DependencyGroups {
			key := strings.ReplaceAll(depGroup.Name, "-", "_")
			depGroupMapping[key] = fmt.Sprintf("(%s)", strings.Join(depGroup.Dependencies, "&&"))
		}
		depExpression, err = translate(conditions, depGroupMapping)
		if err != nil {
			return "", err
		}
	case len(sensor.Spec.DependencyGroups) > 0 && sensor.Spec.DeprecatedCircuit != "" && trigger.Template.DeprecatedSwitch != nil:
		// DEPRECATED.
		logger.Warn("Circuit and Switch are deprecated, please use \"conditions\".")
		temp := ""
		sw := trigger.Template.DeprecatedSwitch
		switch {
		case len(sw.All) > 0:
			temp = strings.Join(sw.All, "&&")
		case len(sw.Any) > 0:
			temp = strings.Join(sw.Any, "||")
		default:
			return "", errors.New("invalid trigger switch")
		}
		groupDepExpr := fmt.Sprintf("(%s) && (%s)", sensor.Spec.DeprecatedCircuit, temp)
		depGroupMapping := make(map[string]string)
		for _, depGroup := range sensor.Spec.DependencyGroups {
			key := strings.ReplaceAll(depGroup.Name, "-", "_")
			depGroupMapping[key] = fmt.Sprintf("(%s)", strings.Join(depGroup.Dependencies, "&&"))
		}
		depExpression, err = translate(groupDepExpr, depGroupMapping)
		if err != nil {
			return "", err
		}
	default:
		deps := []string{}
		for _, dep := range sensor.Spec.Dependencies {
			deps = append(deps, dep.Name)
		}
		depExpression = strings.Join(deps, "&&")
	}
	logger.Sugar().Infof("Dependency expression for trigger %s before simlification: %s", trigger.Template.Name, depExpression)
	boolSimplifier, err := common.NewBoolExpression(depExpression)
	if err != nil {
		logger.Error("Invalid dependency expression", zap.Error(err))
		return "", err
	}
	result := boolSimplifier.GetExpression()
	logger.Sugar().Infof("Dependency expression for trigger %s after simlification: %s", trigger.Template.Name, result)
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
