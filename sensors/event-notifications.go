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
	"github.com/argoproj/argo-events/common"
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/dependencies"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/argoproj/argo-events/sensors/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// isEligibleForExecution determines whether the dependencies are met and triggers are eligible for execution
func isEligibleForExecution(sensor *v1alpha1.Sensor, logger *logrus.Logger) (bool, []string, error) {
	if sensor.Spec.ErrorOnFailedRound && sensor.Status.TriggerCycleStatus == v1alpha1.TriggerCycleFailure {
		return false, nil, errors.Errorf("last trigger cycle was a failure and sensor policy is set to ErrorOnFailedRound, so won't process the triggers")
	}
	if sensor.Spec.Circuit != "" && sensor.Spec.DependencyGroups != nil {
		return dependencies.ResolveCircuit(sensor, logger)
	}
	if ok := snctrl.AreAllDependenciesResolved(sensor); ok {
		var deps []string
		for _, dep := range sensor.Spec.Dependencies {
			deps = append(deps, dep.Name)
		}
		return true, deps, nil
	}
	return false, nil, nil
}

// OperateEventNotifications operates on an event notification
func (sensorCtx *SensorContext) operateEventNotification(notification *types.Notification) error {
	nodeName := notification.EventDependency.Name
	logger := sensorCtx.Logger.WithField(common.LabelEventSource, notification.Event.Context.Source)
	logger.Info("received an event notification")

	snctrl.MarkNodePhase(sensorCtx.Sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, notification.Event, sensorCtx.Logger, "event is received")
	snctrl.MarkUpdatedAt(sensorCtx.Sensor, nodeName)

	// Apply filters
	logger.Infoln("applying filters on event notifications if any")
	if err := dependencies.ApplyFilter(notification); err != nil {
		snctrl.MarkNodePhase(sensorCtx.Sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseError, nil, sensorCtx.Logger, err.Error())
		return err
	}

	// Apply Circuit if any or check if all dependencies are resolved
	logger.Infoln("applying circuit logic if any or checking if all dependencies are resolved")
	ok, snapshot, err := isEligibleForExecution(sensorCtx.Sensor, sensorCtx.Logger)
	if err != nil {
		return err
	}
	if !ok {
		sensorCtx.Logger.Infoln("dependencies are not yet resolved, won't execute triggers")
		return nil
	}

	logger.Infoln("executing triggers")
	// Iterate over each trigger,
	// 1. Apply template level parameters
	// 2. Check if switches are resolved
	// 3. Fetch the resource
	// 4. Apply resource level parameters
	// 5. Execute the trigger
	// 6. If any policy is set, apply it
	for _, trigger := range sensorCtx.Sensor.Spec.Triggers {
		if err := triggers.ApplyTemplateParameters(sensorCtx.Sensor, &trigger); err != nil {
			return err
		}
		if ok := triggers.ApplySwitches(sensorCtx.Sensor, &trigger); !ok {
			logger.Infoln("switches/group level when conditions were not resolved, won't execute the trigger")
			continue
		}

		logger.WithField("trigger-name", trigger.Template.Name).Infoln("resolving the trigger implementation")
		triggerImpl := sensorCtx.GetTrigger(&trigger)
		if triggerImpl == nil {
			logger.WithField("trigger-name", trigger.Template.Name).Errorln("failed to get the specific trigger implementation. continuing to next trigger if any")
			continue
		}

		logger.WithField("trigger-name", trigger.Template.Name).Infoln("fetching trigger resource if any")
		obj, err := triggerImpl.FetchResource()
		if err != nil {
			return err
		}
		if obj == nil {
			logger.WithField("trigger-name", trigger.Template.Name).Warnln("trigger resource is empty")
			continue
		}

		logger.WithField("trigger-name", trigger.Template.Name).Infoln("applying resource parameters if any")
		updatedObj, err := triggerImpl.ApplyResourceParameters(sensorCtx.Sensor, obj)
		if err != nil {
			return err
		}

		logger.WithField("trigger-name", trigger.Template.Name).Infoln("executing the trigger resource")
		newObj, err := triggerImpl.Execute(updatedObj)
		if err != nil {
			return err
		}
		logger.WithField("trigger-name", trigger.Template.Name).Infoln("trigger resource successfully executed")

		logger.WithField("trigger-name", trigger.Template.Name).Infoln("applying trigger policy")
		if err := triggerImpl.ApplyPolicy(newObj); err != nil {
			return err
		}

		logger.WithField("trigger-name", trigger.Template.Name).Infoln("successfully processed the trigger")
	}

	// process snapshot dependencies
	for _, dependency := range snapshot {
		// resolve dependencies
		snctrl.MarkResolvedAt(sensorCtx.Sensor, dependency)
		// Mark snapshot dependency nodes as active
		snctrl.MarkNodePhase(sensorCtx.Sensor, dependency, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, sensorCtx.Logger, "dependency is re-activated")
	}
	// Mark all dependency groups as active
	for _, group := range sensorCtx.Sensor.Spec.DependencyGroups {
		snctrl.MarkNodePhase(sensorCtx.Sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, sensorCtx.Logger, "dependency group is re-activated")
	}

	return nil
}
