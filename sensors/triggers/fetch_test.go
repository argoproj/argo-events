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

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestFetchKubernetesResource(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	artifact := apicommon.NewResource(deployment)
	sensorObj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: &artifact,
	}
	uObj, err := FetchKubernetesResource(fake.NewSimpleClientset(), sensorObj.Spec.Triggers[0].Template.K8s.Source, sensorObj.Namespace)
	assert.Nil(t, err)
	assert.NotNil(t, uObj)
	assert.Equal(t, deployment.GetName(), uObj.GetName())
}
