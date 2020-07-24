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
package slack

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/common/logging"
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
					Slack: &v1alpha1.SlackTrigger{
						SlackToken: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secret",
							},
							Key: "token",
						},
						Namespace: "fake",
						Channel:   "fake-channel",
						Message:   "fake-message",
					},
				},
			},
		},
	},
}

func getSlackTrigger() *SlackTrigger {
	return &SlackTrigger{
		K8sClient:  fake.NewSimpleClientset(),
		Sensor:     sensorObj.DeepCopy(),
		Trigger:    sensorObj.Spec.Triggers[0].DeepCopy(),
		Logger:     logging.NewArgoEventsLogger().Desugar(),
		httpClient: &http.Client{},
	}
}

func TestSlackTrigger_FetchResource(t *testing.T) {
	trigger := getSlackTrigger()
	resource, err := trigger.FetchResource()
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	ot, ok := resource.(*v1alpha1.SlackTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, "fake-channel", ot.Channel)
	assert.Equal(t, "fake-message", ot.Message)
}

func TestSlackTrigger_ApplyResourceParameters(t *testing.T) {
	trigger := getSlackTrigger()

	testEvents := map[string]*v1alpha1.Event{
		"fake-dependency": {
			Context: &v1alpha1.EventContext{
				ID:              "1",
				Type:            "webhook",
				Source:          "webhook-gateway",
				DataContentType: "application/json",
				SpecVersion:     "1.0",
				Subject:         "example-1",
			},
			Data: []byte(`{"channel": "real-channel", "message": "real-message"}`),
		},
	}

	trigger.Trigger.Template.Slack.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "channel",
			},
			Dest: "channel",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "message",
			},
			Dest: "message",
		},
	}

	resource, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.Slack)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	ot, ok := resource.(*v1alpha1.SlackTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, "real-channel", ot.Channel)
	assert.Equal(t, "real-message", ot.Message)
}
