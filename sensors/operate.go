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
)

// operateEventNotification operates on a Event Notification
func (sensorCtx *SensorContext) operateEventNotification(notification *Notification) error {
	nodeName := notification.EventDependency.Name

	logger := sensorCtx.Logger.WithField(common.LabelEventSource, notification.Event.Source())
	logger.Info("received Event Notification")

	// apply filters if any.
	if err := sensorCtx.applyFilter(notification); err != nil {
		snctrl.MarkNodePhase(sensorCtx.Sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseError, nil, sensorCtx.Logger, err.Error())
	}

	snctrl.MarkNodePhase(sensorCtx.Sensor, nodeName, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseComplete, notification.Event, sensorCtx.Logger, "Event is received")

	// check if triggers can be processed and executed
	canProcess, err := sensorCtx.canProcessTriggers()
	if err != nil {
		return err
	}
	if !canProcess {
		return err
	}

	// triggers are ready to process
	sensorCtx.processTriggers()
	return nil
}
