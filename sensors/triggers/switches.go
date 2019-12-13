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

package triggers

import (
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// ApplySwitches applies group level switches on the trigger
func ApplySwitches(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger) bool {
	if trigger.Template.Switch == nil {
		return true
	}
	if trigger.Template.Switch.Any != nil {
		for _, group := range trigger.Template.Switch.Any {
			if status := snctrl.GetNodeByName(sensor, group); status.Type == v1alpha1.NodeTypeDependencyGroup && status.Phase == v1alpha1.NodePhaseComplete {
				return true
			}
		}
		return false
	}
	if trigger.Template.Switch.All != nil {
		for _, group := range trigger.Template.Switch.All {
			if status := snctrl.GetNodeByName(sensor, group); status.Type == v1alpha1.NodeTypeDependencyGroup && status.Phase != v1alpha1.NodePhaseComplete {
				return false
			}
		}
		return true
	}
	return true
}
