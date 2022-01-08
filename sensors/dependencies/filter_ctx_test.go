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
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestFilterContext(t *testing.T) {
	tests := []struct {
		name            string
		expectedContext *v1alpha1.EventContext
		actualContext   *v1alpha1.EventContext
		result          bool
	}{
		{
			name: "different event contexts",
			expectedContext: &v1alpha1.EventContext{
				Type: "webhook",
			},
			actualContext: &v1alpha1.EventContext{
				Type:   "calendar",
				Source: "calendar-gateway",
				ID:     "1",
				Time: metav1.Time{
					Time: time.Now().UTC(),
				},
				DataContentType: common.MediaTypeJSON,
				Subject:         "example-1",
			},
			result: false,
		},
		{
			name: "contexts are same",
			expectedContext: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			actualContext: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Now().UTC(),
				},
				DataContentType: common.MediaTypeJSON,
				Subject:         "example-1",
			},
			result: true,
		},
		{
			name:            "actual event context is nil",
			expectedContext: &v1alpha1.EventContext{},
			actualContext:   nil,
			result:          false,
		},
		{
			name:            "expectedResult event context is nil",
			expectedContext: nil,
			result:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := filterContext(test.expectedContext, test.actualContext)
			assert.Equal(t, test.result, result)
		})
	}
}
