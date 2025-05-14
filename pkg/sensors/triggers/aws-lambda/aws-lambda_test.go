/*
Copyright 2020 The Argoproj Authors.

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
package aws_lambda

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

var sensorObjSparse = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: "fake",
	},
	Spec: v1alpha1.SensorSpec{
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger-sparse",
					AWSLambda: &v1alpha1.AWSLambdaTrigger{
						FunctionName: "fake-function",
						Region:       "us-east",
					},
				},
			},
		},
	},
}

var sensorObjFull = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: "fake",
	},
	Spec: v1alpha1.SensorSpec{
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger-full",
					AWSLambda: &v1alpha1.AWSLambdaTrigger{
						FunctionName: "fake-function",
						AccessKey: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "fake-secret",
							},
							Key: "accesskey",
						},
						SecretKey: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "fake-secret",
							},
							Key: "secretkey",
						},
						Region:  "us-east",
						RoleARN: "arn:aws:iam::123456789012:role/fake-trigger-full",
						Payload: []v1alpha1.TriggerParameter{
							{
								Src: &v1alpha1.TriggerParameterSource{
									DependencyName: "fake-dependency",
									Value:          new(string),
								},
								Dest: "metadata.label.value",
							},
						},
					},
				},
			},
		},
	},
}

func getAWSTriggers() []AWSLambdaTrigger {
	return []AWSLambdaTrigger{
		{
			LambdaClient: nil,
			Sensor:       sensorObjSparse.DeepCopy(),
			Trigger:      &sensorObjSparse.Spec.Triggers[0],
			Logger:       logging.NewArgoEventsLogger(),
		},
		{
			LambdaClient: nil,
			Sensor:       sensorObjFull.DeepCopy(),
			Trigger:      &sensorObjFull.Spec.Triggers[0],
			Logger:       logging.NewArgoEventsLogger(),
		},
	}
}

func TestAWSLambdaTrigger_FetchResource(t *testing.T) {
	triggers := getAWSTriggers()
	for _, trigger := range triggers {
		resource, err := trigger.FetchResource(context.TODO())
		assert.Nil(t, err)
		assert.NotNil(t, resource)

		at, ok := resource.(*v1alpha1.AWSLambdaTrigger)
		assert.Nil(t, err)
		assert.Equal(t, true, ok)
		assert.Equal(t, "fake-function", at.FunctionName)
	}
}

func TestAWSLambdaTrigger_ApplyResourceParameters(t *testing.T) {
	triggers := getAWSTriggers()
	for _, trigger := range triggers {
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
				Data: []byte(`{"function": "real-function"}`),
			},
		}

		defaultValue := "default"
		defaultRegion := "region"

		trigger.Trigger.Template.AWSLambda.Parameters = []v1alpha1.TriggerParameter{
			{
				Src: &v1alpha1.TriggerParameterSource{
					DependencyName: "fake-dependency",
					DataKey:        "function",
					Value:          &defaultValue,
				},
				Dest: "functionName",
			},
			{
				Src: &v1alpha1.TriggerParameterSource{
					DependencyName: "fake-dependency",
					DataKey:        "region",
					Value:          &defaultRegion,
				},
				Dest: "region",
			},
		}

		response, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.AWSLambda)
		assert.Nil(t, err)
		assert.NotNil(t, response)

		updatedObj, ok := response.(*v1alpha1.AWSLambdaTrigger)
		assert.Equal(t, true, ok)
		assert.Equal(t, "real-function", updatedObj.FunctionName)
		assert.Equal(t, "region", updatedObj.Region)
	}
}

func TestAWSLambdaTrigger_ApplyPolicy(t *testing.T) {
	triggers := getAWSTriggers()
	for _, trigger := range triggers {
		status := int64(200)
		response := &lambda.InvokeOutput{
			StatusCode: &status,
		}
		trigger.Trigger.Policy = &v1alpha1.TriggerPolicy{
			Status: &v1alpha1.StatusPolicy{Allow: []int32{200, 300}},
		}
		err := trigger.ApplyPolicy(context.TODO(), response)
		assert.Nil(t, err)
	}
}
