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
	"github.com/argoproj/argo-events/sensors/policy"
	"github.com/argoproj/argo-events/sensors/triggers"
	"github.com/argoproj/argo-events/sensors/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

// isEligibleForExecution determines whether the dependencies are met and triggers are eligible for execution
func isEligibleForExecution(sensor *v1alpha1.Sensor, logger *logrus.Logger) (bool, error) {
	if sensor.Spec.ErrorOnFailedRound && sensor.Status.TriggerCycleStatus == v1alpha1.TriggerCycleFailure {
		return false, errors.Errorf("last trigger cycle was a failure and sensor policy is set to ErrorOnFailedRound, so won't process the triggers")
	}
	if sensor.Spec.Circuit != "" && sensor.Spec.DependencyGroups != nil {
		return dependencies.ResolveCircuit(sensor, logger)
	}
	if ok := sensor.AreAllNodesSuccess(v1alpha1.NodeTypeEventDependency); ok {
		return true, nil
	}
	return false, nil
}

// OperateEventNotifications operates on an event notification
func (sensorCtx *SensorContext) operateEventNotification(notification *types.Notification) error {
	nodeName := notification.EventDependency.Name
	snctrl.MarkNodePhase(sensorCtx.Sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, notification.Event, sensorCtx.Logger, "event is received")

	logger := sensorCtx.Logger.WithField(common.LabelEventSource, notification.Event.Context.Source)
	logger.Info("received an event notification")

	// Apply filters
	logger.Infoln("applying filters on event notifications if any")
	if err := dependencies.ApplyFilter(notification); err != nil {
		snctrl.MarkNodePhase(sensorCtx.Sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseError, nil, sensorCtx.Logger, err.Error())
		return err
	}

	// Apply Circuit if any or check if all dependencies are resolved
	logger.Infoln("applying circuit logic if any or checking if all dependencies are resolved")
	ok, err := isEligibleForExecution(sensorCtx.Sensor, sensorCtx.Logger)
	if err != nil {
		return err
	}
	if !ok {
		sensorCtx.Logger.Infoln("dependencies are not yet resolved, won't execute triggers")
		return nil
	}

	logger.Infoln("starting to execute triggers")
	// Iterate over each trigger,
	// 1. Apply template level parameters
	// 2. Check if switches are resolved
	// 3. Fetch the resource
	// 4. Apply resource level parameters
	// 5. If any policy is set, apply it
	for _, trigger := range sensorCtx.Sensor.Spec.Triggers {
		if err := triggers.ApplyTemplateParameters(sensorCtx.Sensor, &trigger); err != nil {
			return err
		}
		if ok := triggers.ApplySwitches(sensorCtx.Sensor, &trigger); !ok {
			logger.Infoln("switches/group level when conditions were not resolved, won't execute the trigger")
			continue
		}
		uObj, err := triggers.FetchResource(sensorCtx.KubeClient, sensorCtx.Sensor, &trigger)
		if err != nil {
			return err
		}
		if uObj == nil {
			continue
		}
		if err := triggers.ApplyResourceParameters(sensorCtx.Sensor, trigger.ResourceParameters, uObj); err != nil {
			return err
		}
		client := sensorCtx.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    trigger.Template.GroupVersionResource.Group,
			Version:  trigger.Template.GroupVersionResource.Version,
			Resource: trigger.Template.GroupVersionResource.Resource,
		})
		newObj, err := triggers.Execute(sensorCtx.Sensor, uObj, client)
		if err != nil {
			return err
		}
		logger.WithField("trigger-name", trigger.Template.Name).Infoln("trigger successfully created")

		logger.WithField("trigger-name", trigger.Template.Name).Infoln("applying trigger policy")
		p := policy.GetPolicy(&trigger, client, newObj)
		if p == nil {
			logger.WithField("trigger-name", trigger.Template.Name).Infoln("no trigger policy found, continue...")
			continue
		}
		err = p.ApplyPolicy()
		if err != nil {
			switch err {
			case wait.ErrWaitTimeout:
				if trigger.Policy.ErrorOnBackoffTimeout {
					return errors.Errorf("failed to determine status of the triggered resource. setting trigger state as failed")
				}
				continue
			default:
				return err
			}
		}
	}
	return nil
}
