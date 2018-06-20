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

package controller

import (
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// validateSensor accepts a sensor and performs validation against it
func validateSensor(s *v1alpha1.Sensor) error {
	signals := s.Spec.Signals
	triggers := s.Spec.Triggers

	if len(signals) < 1 {
		return fmt.Errorf("No signals found")
	}
	if len(triggers) < 1 {
		return fmt.Errorf("No triggers found")
	}

	for _, signal := range signals {
		// each signal must have a defined type
		if signal.GetType() == "Unknown" {
			return fmt.Errorf("Signal '%s' does not have a defined type", signal.Name)
		}
	}

	for _, trigger := range triggers {
		// each trigger must have a message or a resource
		if trigger.Message == nil && trigger.Resource == nil {
			return fmt.Errorf("trigger '%s' does not contain an absolute action", trigger.Name)
		}
	}

	return nil
}
