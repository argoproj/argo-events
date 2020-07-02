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
package apache_openwhisk

import (
	"net/http"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
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
					OpenWhisk: &v1alpha1.OpenWhiskTrigger{
						Host:       "fake.com",
						ActionName: "hello",
					},
				},
			},
		},
	},
}

func getFakeTriggerImpl() *TriggerImpl {
	return &TriggerImpl{
		OpenWhiskClient: nil,
		K8sClient:       nil,
		Sensor:          sensorObj.DeepCopy(),
		Trigger:         sensorObj.Spec.Triggers[0].DeepCopy(),
		Logger:          common.NewArgoEventsLogger(),
	}
}

func TestTriggerImpl_FetchResource(t *testing.T) {
	trigger := getFakeTriggerImpl()
	obj, err := trigger.FetchResource()
	assert.Nil(t, err)
	assert.NotNil(t, obj)
	trigger1, ok := obj.(*v1alpha1.OpenWhiskTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, trigger.Trigger.Template.OpenWhisk.Host, trigger1.Host)
}

func TestTriggerImpl_ApplyResourceParameters(t *testing.T) {
	trigger := getFakeTriggerImpl()

	testEvents := map[string]*v1alpha1.Event{
		"fake-dependency": {
			Context: &v1alpha1.EventContext{
				ID:              "1",
				Type:            "webhook",
				Source:          "webhook-gateway",
				DataContentType: "application/json",
				SpecVersion:     cloudevents.VersionV1,
				Subject:         "example-1",
			},
			Data: []byte(`{"host": "another-fake.com", "actionName": "world"}`),
		},
	}

	defaultValue := "http://default.com"

	trigger.Trigger.Template.OpenWhisk.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "host",
				Value:          &defaultValue,
			},
			Dest: "host",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "actionName",
				Value:          &defaultValue,
			},
			Dest: "actionName",
		},
	}

	resource, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.OpenWhisk)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	updatedTrigger, ok := resource.(*v1alpha1.OpenWhiskTrigger)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "another-fake.com", updatedTrigger.Host)
}

func TestTriggerImpl_ApplyPolicy(t *testing.T) {
	trigger := getFakeTriggerImpl()
	trigger.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{200, 300}},
	}
	response := &http.Response{StatusCode: 200}
	err := trigger.ApplyPolicy(response)
	assert.Nil(t, err)

	trigger.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{300}},
	}
	err = trigger.ApplyPolicy(response)
	assert.NotNil(t, err)
}
