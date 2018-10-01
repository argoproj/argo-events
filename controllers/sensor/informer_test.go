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

package sensor

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	fake_ss "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/selection"
)

func TestInstanceIDReq(t *testing.T) {
	controller := &SensorController{
		Config: SensorControllerConfig{
			InstanceID: "argo-events",
		},
	}

	req := controller.instanceIDReq()
	assert.Equal(t, common.LabelKeySensorControllerInstanceID, req.Key())
	req = controller.instanceIDReq()
	assert.Equal(t, selection.Equals, req.Operator())
	assert.True(t, req.Values().Has("argo-events"))
}

func TestNewSensorInformer(t *testing.T) {
	controller := &SensorController{
		Config: SensorControllerConfig{
			Namespace:  "testing",
			InstanceID: "argo-events",
		},
		sensorClientset: fake_ss.NewSimpleClientset(),
	}
	controller.newSensorInformer()
}
