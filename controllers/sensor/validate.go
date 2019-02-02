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

package sensor

import (
	"fmt"
	"github.com/Knetic/govaluate"
	"time"

	"github.com/argoproj/argo-events/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// ValidateSensor accepts a sensor and performs validation against it
// we return an error so that it can be logged as a message on the sensor status
// the error is ignored by the operation context as subsequent re-queues would produce the same error.
// Exporting this function so that external APIs can use this to validate sensor resource.
func ValidateSensor(s *v1alpha1.Sensor) error {
	if err := validateSignals(s.Spec.Dependencies); err != nil {
		return err
	}
	err := validateTriggers(s.Spec.Triggers)
	if err != nil {
		return err
	}
	if len(s.Spec.DeploySpec.Containers) > 1 {
		return fmt.Errorf("sensor pod specification can't have more than one container")
	}
	switch s.Spec.EventProtocol.Type {
	case pc.HTTP:
		if s.Spec.EventProtocol.Http.Port == "" {
			return fmt.Errorf("http server port is not defined")
		}
	case pc.NATS:
		if s.Spec.EventProtocol.Nats.URL == "" {
			return fmt.Errorf("nats url is not defined")
		}
		if s.Spec.EventProtocol.Nats.Type == "" {
			return fmt.Errorf("nats type is not defined. either Standard or Streaming type should be defined")
		}
		if s.Spec.EventProtocol.Nats.Type == pc.Streaming && s.Spec.EventProtocol.Nats.ClientId == "" {
			return fmt.Errorf("client id must be specified when using nats streaming")
		}
		if s.Spec.EventProtocol.Nats.Type == pc.Streaming && s.Spec.EventProtocol.Nats.ClusterId == "" {
			return fmt.Errorf("cluster id must be specified when using nats streaming")
		}
	default:
		return fmt.Errorf("unknown gateway type")
	}

	if s.Spec.DependencyGroups != nil {
		if s.Spec.Circuit == "" {
			return fmt.Errorf("no circuit expression provided to resolve dependency groups")
		}
		expression, err := govaluate.NewEvaluableExpression(s.Spec.Circuit)
		if err != nil {
			return fmt.Errorf("circuit expression can't be created for dependency groups. err: %+v", err)
		}

		groups := make(map[string]interface{}, len(s.Spec.DependencyGroups))
		for _, group := range s.Spec.DependencyGroups {
			groups[group.Name] = false
		}
		if _, err = expression.Evaluate(groups); err != nil {
			return fmt.Errorf("circuit expression can't be evaluated for dependency groups. err: %+v", err)
		}
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
		if trigger.Resource == nil {
			return fmt.Errorf("trigger '%s' does not contain an absolute action", trigger.Name)
		}
	}
	return nil
}

// perform a check to see that each event dependency defines one of and at most one of:
// (stream, artifact, calendar, resource, webhook)
func validateSignals(eventDependencies []v1alpha1.EventDependency) error {
	if len(eventDependencies) < 1 {
		return fmt.Errorf("no event dependencies found")
	}
	for _, ed := range eventDependencies {
		if ed.Name == "" {
			return fmt.Errorf("event dependency must define a name")
		}
		if err := validateEventFilter(ed.Filters); err != nil {
			return err
		}
	}
	return nil
}

func validateEventFilter(filter v1alpha1.EventDependencyFilter) error {
	if filter.Time != nil {
		if err := validateEventTimeFilter(filter.Time); err != nil {
			return err
		}
	}
	return nil
}

func validateEventTimeFilter(tFilter *v1alpha1.TimeFilter) error {
	currentT := time.Now().UTC()
	currentT = time.Date(currentT.Year(), currentT.Month(), currentT.Day(), 0, 0, 0, 0, time.UTC)
	currentTStr := currentT.Format(common.StandardYYYYMMDDFormat)
	if tFilter.Start != "" && tFilter.Stop != "" {
		startTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTStr, tFilter.Start))
		if err != nil {
			return err
		}
		stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTStr, tFilter.Stop))
		if err != nil {
			return err
		}
		if stopTime.Before(startTime) || startTime.Equal(stopTime) {
			return fmt.Errorf("invalid event time filter: stop '%s' is before or equal to start '%s", tFilter.Stop, tFilter.Start)
		}
	}
	if tFilter.Stop != "" {
		stopTime, err := time.Parse(common.StandardTimeFormat, fmt.Sprintf("%s %s", currentTStr, tFilter.Stop))
		if err != nil {
			return err
		}
		stopTime = stopTime.UTC()
		if stopTime.Before(currentT.UTC()) {
			return fmt.Errorf("invalid event time filter: stop '%s' is before the current time '%s'", tFilter.Stop, currentT)
		}
	}
	return nil
}
