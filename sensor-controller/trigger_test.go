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

package sensor_controller

import (
	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

var sampleTrigger = v1alpha1.Trigger{
	Name: "sample",
	Resource: &v1alpha1.ResourceObject{
		Namespace: apiv1.NamespaceDefault,
		GroupVersionKind: v1alpha1.GroupVersionKind{
			Group:   "argoproj.io",
			Version: "v1alpha1",
			Kind:    "workflow",
		},
		Source: v1alpha1.ArtifactLocation{
			S3: &v1alpha1.S3Artifact{},
		},
		Labels: map[string]string{"test-label": "test-value"},
	},
}
