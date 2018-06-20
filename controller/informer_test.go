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

package controller

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	fake_ss "github.com/argoproj/argo-events/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInstanceIDReq(t *testing.T) {
	controller := &SensorController{
		Config: SensorControllerConfig{
			InstanceID: "",
		},
	}

	req := controller.instanceIDReq()
	assert.Equal(t, common.LabelKeySensorControllerInstanceID, req.Key())
	assert.Equal(t, selection.DoesNotExist, req.Operator())

	controller.Config.InstanceID = "axis"
	req = controller.instanceIDReq()
	assert.Equal(t, selection.Equals, req.Operator())
	assert.True(t, req.Values().Has("axis"))
}

func TestNewSensorInformer(t *testing.T) {
	controller := &SensorController{
		Config: SensorControllerConfig{
			Namespace:  "testing",
			InstanceID: "",
		},
		sensorClientset: fake_ss.NewSimpleClientset(),
	}
	controller.newSensorInformer()
}

func TestNewPodInformer(t *testing.T) {
	controller := &SensorController{
		Config: SensorControllerConfig{
			Namespace:  "testing",
			InstanceID: "",
		},
		kubeClientset: fake.NewSimpleClientset(),
	}
	controller.newPodInformer()
}
