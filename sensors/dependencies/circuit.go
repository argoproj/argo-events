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

package dependencies

import (
	"github.com/Knetic/govaluate"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// ResolveCircuit resolves a circuit
func ResolveCircuit(sensor *v1alpha1.Sensor, logger *logrus.Logger) (bool, []string, error) {
	if sensor.Spec.DependencyGroups != nil {
		groups := make(map[string]interface{}, len(sensor.Spec.DependencyGroups))
	group:
		for _, group := range sensor.Spec.DependencyGroups {
			for _, dependency := range group.Dependencies {
				if resolved := snctrl.IsDependencyResolved(sensor, dependency); !resolved {
					groups[group.Name] = false
					continue group
				}
			}
			snctrl.MarkNodePhase(sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseComplete, nil, logger, "dependency group is complete")
			groups[group.Name] = true
		}

		expression, err := govaluate.NewEvaluableExpression(sensor.Spec.Circuit)
		if err != nil {
			return false, nil, errors.Errorf("failed to create circuit expression. err: %+v", err)
		}

		result, err := expression.Evaluate(groups)
		if err != nil {
			return false, nil, errors.Errorf("failed to evaluate circuit expression. err: %+v", err)
		}
		if result != true {
			return false, nil, nil
		}

		var snapshot []string
		for name, resolved := range groups {
			if resolved == true {
				for _, depGroups := range sensor.Spec.DependencyGroups {
					if depGroups.Name == name {
						snapshot = append(snapshot, depGroups.Dependencies...)
					}
				}
			}
		}

		return true, snapshot, err
	}
	return false, nil, nil
}
