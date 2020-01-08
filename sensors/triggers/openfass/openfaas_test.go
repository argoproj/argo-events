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
package openfass

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var sensorObj = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: "fake",
	},
	Spec: v1alpha1.SensorSpec{
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					OpenFaas: &v1alpha1.OpenFaasTrigger{
						GatewayURL: "http://openfaas-fake.com",
						Password: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secret",
							},
							Key: "password",
						},
						Namespace:    "fake",
						FunctionName: "fake-function",
					},
				},
			},
		},
	},
}

func getOpenFaasTrigger() *OpenFaasTrigger {
	return NewOpenFaasTrigger(fake.NewSimpleClientset(), sensorObj.DeepCopy(), sensorObj.Spec.Triggers[0].DeepCopy(), common.NewArgoEventsLogger())
}

func TestOpenFaasTrigger_FetchResource(t *testing.T) {
	trigger := getOpenFaasTrigger()
	resource, err := trigger.FetchResource()
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	ot, ok := resource.(*v1alpha1.OpenFaasTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, "fake-function", ot.FunctionName)
}

func TestOpenFaasTrigger_ApplyResourceParameters(t *testing.T) {
	trigger := getOpenFaasTrigger()
	id := trigger.Sensor.NodeID("fake-dependency")
	trigger.Sensor.Status = v1alpha1.SensorStatus{
		Nodes: map[string]v1alpha1.NodeStatus{
			id: {
				Name: "fake-dependency",
				Type: v1alpha1.NodeTypeEventDependency,
				ID:   id,
				Event: &apicommon.Event{
					Context: apicommon.EventContext{
						ID:              "1",
						Type:            "webhook",
						Source:          "webhook-gateway",
						DataContentType: "application/json",
						SpecVersion:     "0.3",
						Subject:         "example-1",
					},
					Data: []byte(`{"gateway-url": "http://another-fake.com", "function-name": "real-function"}`),
				},
			},
		},
	}

	trigger.Trigger.Template.OpenFaas.ResourceParameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "gateway-url",
			},
			Dest: "gatewayURL",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "function-name",
			},
			Dest: "functionName",
		},
	}

	resource, err := trigger.ApplyResourceParameters(trigger.Sensor, trigger.Trigger.Template.OpenFaas)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	ot, ok := resource.(*v1alpha1.OpenFaasTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, "real-function", ot.FunctionName)
	assert.Equal(t, "http://another-fake.com", ot.GatewayURL)
}
