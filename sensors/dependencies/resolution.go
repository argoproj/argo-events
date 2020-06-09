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
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"
)

// ResolveDependency resolves a dependency based on Event and gateway name
func ResolveDependency(dependencies []v1alpha1.EventDependency, event *v1alpha1.Event, logger *logrus.Logger) *v1alpha1.EventDependency {
	for _, dependency := range dependencies {
		eventNameGlob, err := glob.Compile(dependency.EventName)
		if err != nil {
			continue
		}
		if !eventNameGlob.Match(event.Context.Subject) {
			continue
		}
		var sourceGlob glob.Glob
		if len(dependency.EventSourceName) > 0 {
			sourceGlob, err = glob.Compile(dependency.EventSourceName)
		} else if len(dependency.GatewayName) > 0 {
			// DEPRECATED:
			logger.WithFields(logrus.Fields{
				"source":  event.Context.Source,
				"subject": event.Context.Subject,
			}).Warn("spec.dependencies.gatewayName is DEPRECATED, it will be unsupported soon, please use spec.dependencies.eventSourceName")
			sourceGlob, err = glob.Compile(dependency.GatewayName)
		} else {
			continue
		}
		if err != nil {
			continue
		}
		if !sourceGlob.Match(event.Context.Source) {
			continue
		}
		return &dependency
	}
	return nil
}
