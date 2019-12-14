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

package types

import (
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// Notification to update event dependency's state or the sensor resource
type Notification struct {
	// Event is the internal representation of cloud event received from the gateway
	Event *apicommon.Event
	// EventDependency refers to the dependency against the event received from teh gateway
	EventDependency *v1alpha1.EventDependency
	// Sensor refers to the sensor object
	// It is a reference to the sensor object updated by some other process and now it is required
	// that the sensor context must update it's own object reference
	Sensor *v1alpha1.Sensor
	// NotificationType for event notification and state update notification
	NotificationType v1alpha1.NotificationType
}
