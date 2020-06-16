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
	"github.com/gobwas/glob"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// ResolveDependency resolves a dependency based on Event and gateway name
func ResolveDependency(dependencies []v1alpha1.EventDependency, events *v1alpha1.Event) *v1alpha1.EventDependency {
	for _, dependency := range dependencies {
		gatewayNameGlob, err := glob.Compile(dependency.GatewayName)
		if err != nil {
			continue
		}
		eventNameGlob, err := glob.Compile(dependency.EventName)
		if err != nil {
			continue
		}
		if gatewayNameGlob.Match(events.Context.Source) && eventNameGlob.Match(events.Context.Subject) {
			return &dependency
		}
	}
	return nil
}
