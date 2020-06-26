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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestResolveDependency(t *testing.T) {
	obj := sensorObj.DeepCopy()

	tests := []struct {
		name       string
		updateFunc func()
		testFunc   func(dep *v1alpha1.EventDependency)
	}{
		{
			name: "unknown dependency",
			updateFunc: func() {
				obj.Spec.Dependencies[0].GatewayName = "fake-source"
				obj.Spec.Dependencies[0].EventName = "example"
			},
			testFunc: func(dep *v1alpha1.EventDependency) {
				assert.Nil(t, dep)
			},
		},
		{
			name: "known dependency",
			updateFunc: func() {
				obj.Spec.Dependencies[0].GatewayName = "fake-source"
				obj.Spec.Dependencies[0].EventName = "example-1"
			},
			testFunc: func(dep *v1alpha1.EventDependency) {
				assert.NotNil(t, dep)
			},
		},
		{
			name: "glob pattern",
			updateFunc: func() {
				obj.Spec.Dependencies[0].GatewayName = "fake-*"
				obj.Spec.Dependencies[0].EventName = "example-*"
			},
			testFunc: func(dep *v1alpha1.EventDependency) {
				assert.NotNil(t, dep)
			},
		},
	}

	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			Source:  "fake-source",
			Subject: "example-1",
		},
		Data: nil,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.updateFunc()
			dependency := ResolveDependency(obj.Spec.Dependencies, event, common.NewArgoEventsLogger())
			test.testFunc(dependency)
		})
	}
}
