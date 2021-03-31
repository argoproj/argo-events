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
package azure_event_hubs

import (
	"context"
	"testing"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					AzureEventHubs: &v1alpha1.AzureEventHubsTrigger{
						FQDN:       "fake-url",
						HubName:    "fake-hub",
						Parameters: nil,
						SharedAccessKey: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "fake-secret",
							},
							Key: "sharedAccessKey",
						},
						SharedAccessKeyName: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "fake-secret",
							},
							Key: "sharedAccessKeyName",
						},
						Payload: nil,
					},
				},
			},
		},
	},
}

func getFakeAzureEventHubsTrigger() *AzureEventHubsTrigger {
	return &AzureEventHubsTrigger{
		Hub:     nil,
		Sensor:  sensorObj.DeepCopy(),
		Trigger: &sensorObj.Spec.Triggers[0],
		Logger:  logging.NewArgoEventsLogger(),
	}
}

func TestAzureEventHubsTrigger_FetchResource(t *testing.T) {
	trigger := getFakeAzureEventHubsTrigger()
	resource, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	at, ok := resource.(*v1alpha1.AzureEventHubsTrigger)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "fake-url", at.FQDN)
	assert.Equal(t, "fake-hub", at.HubName)
}

func TestAzureEventHubsTrigger_ApplyResourceParameters(t *testing.T) {
	trigger := getFakeAzureEventHubsTrigger()

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
			Data: []byte(`{"fqdn": "real-fqdn"}`),
		},
	}

	defaultValue := "fake-azure-event-hubs-fqdn"

	trigger.Trigger.Template.AzureEventHubs.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "fqdn",
				Value:          &defaultValue,
			},
			Dest: "fqdn",
		},
	}

	response, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.AzureEventHubs)
	assert.Nil(t, err)
	assert.NotNil(t, response)

	updatedTrigger, ok := response.(*v1alpha1.AzureEventHubsTrigger)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "real-fqdn", updatedTrigger.FQDN)
}
