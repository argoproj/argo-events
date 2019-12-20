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
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ResolveCircuit resolves a circuit
func ResolveCircuit(sensor *v1alpha1.Sensor, logger *logrus.Logger) (bool, error) {
	if sensor.Spec.DependencyGroups != nil {
		groups := make(map[string]interface{}, len(sensor.Spec.DependencyGroups))
	group:
		for _, group := range sensor.Spec.DependencyGroups {
			for _, dependency := range group.Dependencies {
				if nodeStatus := snctrl.GetNodeByName(sensor, dependency); nodeStatus.Phase != v1alpha1.NodePhaseComplete {
					groups[group.Name] = false
					continue group
				}
			}
			snctrl.MarkNodePhase(sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseComplete, nil, logger, "dependency group is complete")
			groups[group.Name] = true
		}

		expression, err := govaluate.NewEvaluableExpression(sensor.Spec.Circuit)
		if err != nil {
			return false, errors.Errorf("failed to create circuit expression. err: %+v", err)
		}

		result, err := expression.Evaluate(groups)
		if err != nil {
			return false, errors.Errorf("failed to evaluate circuit expression. err: %+v", err)
		}

		return result == true, err
	}
	return false, nil
}
