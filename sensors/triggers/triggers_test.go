/*
Copyright 2020 BlackRock, Inc.

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

package triggers

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/stretchr/testify/assert"
)

func TestGetGroupVersionResource(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test-deployment")
	expectedDeploymentGVR := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	assert.Equal(t, expectedDeploymentGVR, GetGroupVersionResource(deployment))

	ingress := newUnstructured("networking.k8s.io/v1", "Ingress", "fake", "test-ingress")
	expectedIngressGVR := schema.GroupVersionResource{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	}
	assert.Equal(t, expectedIngressGVR, GetGroupVersionResource(ingress))

	eventbus := newUnstructured("argoproj.io/v1alpha1", "EventBus", "fake", "test-eb")
	expectedEventBusGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "eventbus",
	}
	assert.Equal(t, expectedEventBusGVR, GetGroupVersionResource(eventbus))
}
