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
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/gobwas/glob"
)

// resolveDependency resolves a dependency based on event and gateway name
func (sensorCtx *sensorContext) resolveDependency(events *cloudevents.Event) *v1alpha1.EventDependency {
	for _, dependency := range sensorCtx.sensor.Spec.Dependencies {
		gatewayNameGlob, err := glob.Compile(dependency.GatewayName)
		if err != nil {
			sensorCtx.logger.WithError(err).WithField("gateway-name", dependency.GatewayName).Error("invalid glob in gateway name")
			continue
		}

		eventNameGlob, err := glob.Compile(dependency.EventName)
		if err != nil {
			sensorCtx.logger.WithError(err).Error("invalid glob in event name")
			continue
		}

		if gatewayNameGlob.Match(events.Context.GetSource()) && eventNameGlob.Match(events.Context.GetSubject()) {
			return &dependency
		}
	}
	return nil
}
