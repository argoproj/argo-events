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
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/operator"
)

// processQueue processes events received on internal queue and updates the state of the node representing the event dependency
func (sensorCtx *SensorContext) processQueue(notification *Notification) {
	defer func() {
		updatedSensor, err := snctrl.PersistUpdates(sensorCtx.SensorClient, sensorCtx.Sensor, sensorCtx.Logger)
		if err != nil {
			sensorCtx.Logger.WithError(err).Error("failed to persist sensor update")
		}
		// update Sensor ref. in case of failure to persist updates, this is a deep copy of old Sensor resource
		sensorCtx.Sensor = updatedSensor
	}()

	switch notification.NotificationType {
	case v1alpha1.EventNotification:
		if err := operator.OperateEventNotifications(sensorCtx, notification); err != nil {
			sensorCtx.Logger.WithError(err).Errorln("failed to operate on the event notification")
		}

	case v1alpha1.ResourceUpdateNotification:
		operator.OperateResourceUpdateNotifications(sensorCtx, notification)

	default:
		sensorCtx.Logger.WithField("Notification-type", string(notification.NotificationType)).Error("unknown Notification type")
	}
}
