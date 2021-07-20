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
	"math/rand"
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
	"github.com/argoproj/argo-events/common/leaderelection"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
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

func (sensorCtx *SensorContext) getGroupAndClientID(depExpression string) (string, string) {
	// Generate clientID with hash code
	hashKey := fmt.Sprintf("%s-%s", sensorCtx.sensor.Name, depExpression)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	hashVal := common.Hasher(hashKey)
	group := fmt.Sprintf("client-%v", hashVal)
	clientID := fmt.Sprintf("client-%v-%v", hashVal, r1.Intn(100))
	return group, clientID
}

func (sensorCtx *SensorContext) Start(ctx context.Context) error {
	log := logging.FromContext(ctx)
	custerName := fmt.Sprintf("%s-sensor-%s", sensorCtx.sensor.Namespace, sensorCtx.sensor.Name)
	elector, err := leaderelection.NewEventBusElector(ctx, *sensorCtx.eventBusConfig, custerName, int(sensorCtx.sensor.Spec.GetReplicas()))
	if err != nil {
		log.Errorw("failed to get an elector", zap.Error(err))
		return err
	}
	elector.RunOrDie(ctx, leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			if err := sensorCtx.listenEvents(ctx); err != nil {
				log.Errorw("failed to start", zap.Error(err))
			}
		},
		OnStoppedLeading: func() {
			log.Infof("leader lost: %s", sensorCtx.hostname)
		},
	})
	return nil
}

// listenEvents watches and handles events received from the gateway.
func (sensorCtx *SensorContext) listenEvents(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	sensor := sensorCtx.sensor
	// Get a mapping of dependencyExpression: []triggers
	triggerMapping := make(map[string][]v1alpha1.Trigger)
	for _, trigger := range sensor.Spec.Triggers {
		depExpr, err := sensorCtx.getDependencyExpression(ctx, trigger)
		if err != nil {
			logger.Errorw("failed to get dependency expression", zap.Error(err))
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
				logger.Errorw("failed to get new evaluable expression", zap.Error(err))
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
				d := eventbusdriver.Dependency{
					Name:            dep.Name,
					EventSourceName: dep.EventSourceName,
					EventName:       dep.EventName,
				}
				deps = append(deps, d)
			}

			group, clientID := sensorCtx.getGroupAndClientID(depExpression)
			ebDriver, err := eventbus.GetDriver(cctx, *sensorCtx.eventBusConfig, sensorCtx.eventBusSubject, clientID)
			if err != nil {
				logger.Errorw("failed to get eventbus driver", zap.Error(err))
				return
			}
			triggerNames := []string{}
			for _, t := range triggers {
				triggerNames = append(triggerNames, t.Template.Name)
			}
			var conn eventbusdriver.Connection
			err = common.Connect(&common.DefaultBackoff, func() error {
				var err error
				conn, err = ebDriver.Connect()
				return err
			})
			if err != nil {
				logger.Fatalw("failed to connect to event bus", zap.Error(err))
				return
			}
			defer conn.Close()

			transformFunc := func(depName string, event *cloudevents.Event) (*cloudevents.Event, error) {
				dep, ok := depMapping[depName]
				if !ok {
					return nil, fmt.Errorf("dependency not found")
				}
				if dep.JQExpr == "" {
					return event, nil
				}
				transformedEvent, err := sensordependencies.Transform(event, dep.JQExpr)
				if err != nil {
					logger.Errorw("failed to apply jq transformation", zap.Error(err))
					return nil, err
				}
				return transformedEvent, nil
			}

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
					logger.Errorw("failed to apply filters", zap.Error(err))
					return false
				}
				return result
			}

			actionFunc := func(events map[string]cloudevents.Event) {
				if err := sensorCtx.triggerActions(cctx, sensor, events, triggers); err != nil {
					logger.Errorw("failed to trigger actions", zap.Error(err))
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

					logger.Infof("started subscribing to events for triggers %s with client %s", fmt.Sprintf("[%s]", strings.Join(triggerNames, " ")), clientID)

					err = ebDriver.SubscribeEventSources(cctx, conn, group, closeSubCh, depExpression, deps, transformFunc, filterFunc, actionFunc)
					if err != nil {
						logger.Errorw("failed to subscribe to eventbus", zap.Any("clientID", clientID), zap.Error(err))
						return
					}
				}()
			}

			subscribeOnce(&subLock, subscribeFunc)

			logger.Infof("starting eventbus connection daemon for client %s...", clientID)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-cctx.Done():
					logger.Infof("exiting eventbus connection daemon for client %s...", clientID)
					wg1.Wait()
					return
				case <-ticker.C:
					if conn == nil || conn.IsClosed() {
						logger.Info("NATS connection lost, reconnecting...")
						// Regenerate the client ID to avoid the issue that NAT server still thinks the client is alive.
						_, clientID := sensorCtx.getGroupAndClientID(depExpression)
						ebDriver, err := eventbus.GetDriver(cctx, *sensorCtx.eventBusConfig, sensorCtx.eventBusSubject, clientID)
						if err != nil {
							logger.Errorw("failed to get eventbus driver during reconnection", zap.Error(err))
							continue
						}
						conn, err = ebDriver.Connect()
						if err != nil {
							logger.Errorw("failed to reconnect to eventbus", zap.Any("clientID", clientID), zap.Error(err))
							continue
						}
						logger.Infow("reconnected to NATS streaming server.", zap.Any("clientID", clientID))

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
	<-ctx.Done()
	logger.Info("Shutting down...")
	cancel()
	wg.Wait()
	return nil
}

func (sensorCtx *SensorContext) triggerActions(ctx context.Context, sensor *v1alpha1.Sensor, events map[string]cloudevents.Event, triggers []v1alpha1.Trigger) error {
	log := logging.FromContext(ctx)
	eventsMapping := make(map[string]*v1alpha1.Event)
	depNames := make([]string, 0, len(events))
	eventIDs := make([]string, 0, len(events))
	for k, v := range events {
		eventsMapping[k] = convertEvent(v)
		depNames = append(depNames, k)
		eventIDs = append(eventIDs, v.ID())
	}
	for _, trigger := range triggers {
		if err := sensorCtx.triggerOne(ctx, sensor, trigger, eventsMapping, depNames, eventIDs, log); err != nil {
			// Log the error, and let it continue
			log.Errorw("failed to execute a trigger", zap.Error(err), zap.String(logging.LabelTriggerName, trigger.Template.Name),
				zap.Any("triggeredBy", depNames), zap.Any("triggeredByEvents", eventIDs))
			sensorCtx.metrics.ActionFailed(sensor.Name, trigger.Template.Name)
		} else {
			sensorCtx.metrics.ActionTriggered(sensor.Name, trigger.Template.Name)
		}
	}
	return nil
}

func (sensorCtx *SensorContext) triggerOne(ctx context.Context, sensor *v1alpha1.Sensor, trigger v1alpha1.Trigger, eventsMapping map[string]*v1alpha1.Event, depNames, eventIDs []string, log *zap.SugaredLogger) error {
	defer func(start time.Time) {
		sensorCtx.metrics.ActionDuration(sensor.Name, trigger.Template.Name, float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	defer sensorCtx.metrics.ActionDuration(sensor.Name, trigger.Template.Name, float64(time.Since(time.Now())/time.Millisecond))

	if err := sensortriggers.ApplyTemplateParameters(eventsMapping, &trigger); err != nil {
		log.Errorf("failed to apply template parameters, %v", err)
		return err
	}

	logger := log.With(logging.LabelTriggerName, trigger.Template.Name)

	logger.Debugw("resolving the trigger implementation")
	triggerImpl := sensorCtx.GetTrigger(ctx, &trigger)
	if triggerImpl == nil {
		return errors.Errorf("invalid trigger %s, could not find an implementation", trigger.Template.Name)
	}

	logger = logger.With(logging.LabelTriggerType, triggerImpl.GetTriggerType())
	log.Debug("fetching trigger resource if any")
	obj, err := triggerImpl.FetchResource(ctx)
	if err != nil {
		return err
	}
	if obj == nil {
		return errors.Errorf("invalid trigger %s, could not fetch the trigger resource", trigger.Template.Name)
	}

	logger.Debug("applying resource parameters if any")
	updatedObj, err := triggerImpl.ApplyResourceParameters(eventsMapping, obj)
	if err != nil {
		return err
	}

	logger.Debug("executing the trigger resource")
	retryStrategy := trigger.RetryStrategy
	if retryStrategy == nil {
		retryStrategy = &apicommon.Backoff{Steps: 1}
	}
	var newObj interface{}
	if err := common.Connect(retryStrategy, func() error {
		var e error
		newObj, e = triggerImpl.Execute(ctx, eventsMapping, updatedObj)
		return e
	}); err != nil {
		return errors.Wrap(err, "failed to execute trigger")
	}
	logger.Debug("trigger resource successfully executed")

	logger.Debug("applying trigger policy")
	if err := triggerImpl.ApplyPolicy(ctx, newObj); err != nil {
		return err
	}
	logger.Infow("successfully processed the trigger",
		zap.Any("triggeredBy", depNames), zap.Any("triggeredByEvents", eventIDs))
	return nil
}

func (sensorCtx *SensorContext) getDependencyExpression(ctx context.Context, trigger v1alpha1.Trigger) (string, error) {
	logger := logging.FromContext(ctx)

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
			logger.Errorw("Failed to compile original dependency expression", zap.Error(err))
			return "", err
		}
		result, err := expr.Run(program, parameters)
		if err != nil {
			logger.Errorw("Failed to parse original dependency expression", zap.Error(err))
			return "", err
		}
		newExpr := fmt.Sprintf("%v", result)
		newExpr = strings.ReplaceAll(newExpr, "\"(\"", "(")
		newExpr = strings.ReplaceAll(newExpr, "\")\"", ")")
		return newExpr, nil
	}

	sensor := sensorCtx.sensor
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
	logger.Infof("Dependency expression for trigger %s before simplification: %s", trigger.Template.Name, depExpression)
	boolSimplifier, err := common.NewBoolExpression(depExpression)
	if err != nil {
		logger.Errorw("Invalid dependency expression", zap.Error(err))
		return "", err
	}
	result := boolSimplifier.GetExpression()
	logger.Infof("Dependency expression for trigger %s after simplification: %s", trigger.Template.Name, result)
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
