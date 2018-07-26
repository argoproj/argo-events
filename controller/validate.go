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
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateSensor accepts a sensor and performs validation against it
// we return an error so that it can be logged as a message on the sensor status
// the error is ignored by the operation context as subsequent re-queues would produce the same error.
func validateSensor(s *v1alpha1.Sensor) error {
	if err := validateSignals(s.Spec.Signals); err != nil {
		return err
	}
	if err := validateTriggers(s.Spec.Triggers); err != nil {
		return err
	}
	return nil
}

func validateTriggers(triggers []v1alpha1.Trigger) error {
	if len(triggers) < 1 {
		return fmt.Errorf("no triggers found")
	}

	for _, trigger := range triggers {
		if trigger.Name == "" {
			return fmt.Errorf("trigger must define a name")
		}
		// each trigger must have a message or a resource
		if trigger.Message == nil && trigger.Resource == nil {
			return fmt.Errorf("trigger '%s' does not contain an absolute action", trigger.Name)
		}
	}
	return nil
}

// perform a check to see that each signal defines one of and at most one of:
// (stream, artifact, calendar, resource, webhook)
func validateSignals(signals []v1alpha1.Signal) error {
	if len(signals) < 1 {
		return fmt.Errorf("no signals found")
	}
	for _, signal := range signals {
		if signal.Name == "" {
			return fmt.Errorf("signal must define a name")
		}
		// each signal must have one and at most one defined type
		if signal.GetType() == "Unknown" {
			return fmt.Errorf("signal '%s' does not have a defined type", signal.Name)
		}
		signalErrs := make(map[v1alpha1.SignalType]error)
		i := 0
		if signal.Stream != nil {
			if err := validateStreamSignal(signal.Stream); err != nil {
				signalErrs[v1alpha1.SignalTypeStream] = err
			}
			i++
		}
		if signal.Artifact != nil {
			if err := validateArtifactSignal(signal.Artifact); err != nil {
				signalErrs[v1alpha1.SignalTypeArtifact] = err
			}
			i++
		}
		if signal.Calendar != nil {
			if err := validateCalendarSignal(signal.Calendar); err != nil {
				signalErrs[v1alpha1.SignalTypeCalendar] = err
			}
			i++
		}
		if signal.Resource != nil {
			//todo: validate resource signals
			i++
		}
		if signal.Webhook != nil {
			//todo: validate webhook signals
			i++
		}
		if i != 1 {
			return fmt.Errorf("signal '%s' defines multiple types", signal.Name)
		}
		if len(signalErrs) > 0 {
			return fmt.Errorf("%v", signalErrs)
		}

		if err := validateSignalFilter(signal.Filters); err != nil {
			return err
		}
	}
	return nil
}

func validateStreamSignal(stream *v1alpha1.Stream) error {
	if stream.Type == "" {
		return fmt.Errorf("invalid stream: type should not be empty")
	}
	if stream.URL == "" {
		return fmt.Errorf("invalid stream: URL should not be empty")
	}
	return nil
}

func validateArtifactSignal(artifact *v1alpha1.ArtifactSignal) error {
	if !artifact.HasLocation() {
		return fmt.Errorf("invalid artifact signal: source location is missing")
	}
	if err := validateStreamSignal(&artifact.Target); err != nil {
		return fmt.Errorf("invalid artifact signal: target stream failed with %s", err)
	}
	return nil
}

func validateCalendarSignal(calendar *v1alpha1.CalendarSignal) error {
	if calendar.Interval == "" && calendar.Schedule == "" {
		return fmt.Errorf("invalid calendar signal: one of interval or schedule should be specified")
	}
	if _, err := common.ParseExclusionDates(calendar.Recurrence); err != nil {
		return err
	}
	return nil
}

func validateSignalFilter(filter v1alpha1.SignalFilter) error {
	if filter.Time != nil {
		if err := validateSignalTimeFilter(filter.Time); err != nil {
			return err
		}
	}
	return nil
}

func validateSignalTimeFilter(tFilter *v1alpha1.TimeFilter) error {
	currentT := metav1.Time{Time: time.Now().UTC()}
	if tFilter.Start != nil && tFilter.Stop != nil {
		if tFilter.Stop.Before(tFilter.Start) || tFilter.Start.Equal(tFilter.Stop) {
			return fmt.Errorf("invalid signal time filter: stop '%s' is before or equal to start '%s", tFilter.Stop, tFilter.Start)
		}
	}
	if tFilter.Stop != nil {
		if tFilter.Stop.Before(&currentT) {
			return fmt.Errorf("invalid signal time filter: stop '%s' is before the current time '%s'", tFilter.Stop, currentT)
		}
	}
	return nil
}
