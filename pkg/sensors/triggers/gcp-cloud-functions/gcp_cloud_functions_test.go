/*
Copyright 2026 The Argoproj Authors.

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
package gcpcloudfunctions

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
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
					GCPCloudFunctions: &v1alpha1.GCPCloudFunctionsTrigger{
						URL: "https://my-function-1234567890.us-central1.run.app",
						Payload: []v1alpha1.TriggerParameter{
							{
								Src: &v1alpha1.TriggerParameterSource{
									DependencyName: "fake-dependency",
									DataKey:        "message",
								},
								Dest: "message",
							},
						},
					},
				},
			},
		},
	},
}

func getGCPTrigger() GCPCloudFunctionsTrigger {
	return GCPCloudFunctionsTrigger{
		Client:  nil,
		Sensor:  sensorObj.DeepCopy(),
		Trigger: &sensorObj.Spec.Triggers[0],
		Logger:  logging.NewArgoEventsLogger(),
	}
}

func TestGCPCloudFunctionsTrigger_GetTriggerType(t *testing.T) {
	trigger := getGCPTrigger()
	assert.Equal(t, v1alpha1.TriggerTypeGCPCloudFunctions, trigger.GetTriggerType())
}

func TestGCPCloudFunctionsTrigger_FetchResource(t *testing.T) {
	trigger := getGCPTrigger()

	resource, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	gt, ok := resource.(*v1alpha1.GCPCloudFunctionsTrigger)
	assert.True(t, ok)
	assert.Equal(t, "https://my-function-1234567890.us-central1.run.app", gt.URL)
	assert.Equal(t, http.MethodPost, gt.Method) // default
}

func TestGCPCloudFunctionsTrigger_ApplyResourceParameters(t *testing.T) {
	trigger := getGCPTrigger()

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
			Data: []byte(`{"url": "https://new-function-9876543210.europe-west1.run.app"}`),
		},
	}

	defaultValue := "default"
	trigger.Trigger.Template.GCPCloudFunctions.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "url",
				Value:          &defaultValue,
			},
			Dest: "url",
		},
	}

	response, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.GCPCloudFunctions)
	assert.Nil(t, err)
	assert.NotNil(t, response)

	updatedObj, ok := response.(*v1alpha1.GCPCloudFunctionsTrigger)
	assert.True(t, ok)
	assert.Equal(t, "https://new-function-9876543210.europe-west1.run.app", updatedObj.URL)
}

func TestGCPCloudFunctionsTrigger_ApplyPolicy(t *testing.T) {
	trigger := getGCPTrigger()

	response := &http.Response{
		StatusCode: 200,
	}
	trigger.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{200, 201}},
	}
	err := trigger.ApplyPolicy(context.TODO(), response)
	assert.Nil(t, err)
}

func TestGCPCloudFunctionsTrigger_ApplyPolicyFail(t *testing.T) {
	trigger := getGCPTrigger()

	response := &http.Response{
		StatusCode: 500,
	}
	trigger.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{200, 201}},
	}
	err := trigger.ApplyPolicy(context.TODO(), response)
	assert.NotNil(t, err)
}

func TestGCPCloudFunctionsTrigger_ApplyPolicyNil(t *testing.T) {
	trigger := getGCPTrigger()
	trigger.Trigger.Policy = nil
	err := trigger.ApplyPolicy(context.TODO(), &http.Response{StatusCode: 200})
	assert.Nil(t, err)
}

// fakeRoundTripper captures the request and returns a canned response.
type fakeRoundTripper struct {
	capturedReq *http.Request
	statusCode  int
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.capturedReq = req
	return &http.Response{
		StatusCode: f.statusCode,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}, nil
}

func TestGCPCloudFunctionsTrigger_Execute(t *testing.T) {
	rt := &fakeRoundTripper{statusCode: 200}
	trigger := getGCPTrigger()
	trigger.Client = &http.Client{Transport: rt}

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
			Data: []byte(`{"message": "hello"}`),
		},
	}

	resource, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)

	result, err := trigger.Execute(context.TODO(), testEvents, resource)
	assert.Nil(t, err)
	assert.NotNil(t, result)

	resp, ok := result.(*http.Response)
	assert.True(t, ok)
	assert.Equal(t, 200, resp.StatusCode)
	assert.NotNil(t, rt.capturedReq)
	assert.Equal(t, http.MethodPost, rt.capturedReq.Method)
	assert.Equal(t, "https://my-function-1234567890.us-central1.run.app", rt.capturedReq.URL.String())
	assert.Equal(t, "application/json", rt.capturedReq.Header.Get("Content-Type"))
}

func TestGCPCloudFunctionsTrigger_Execute_CustomHeaders(t *testing.T) {
	rt := &fakeRoundTripper{statusCode: 200}
	trigger := getGCPTrigger()
	trigger.Client = &http.Client{Transport: rt}
	trigger.Trigger.Template.GCPCloudFunctions.Headers = map[string]string{
		"X-Custom-Header": "custom-value",
	}

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
			Data: []byte(`{"message": "hello"}`),
		},
	}

	resource, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)

	result, err := trigger.Execute(context.TODO(), testEvents, resource)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "custom-value", rt.capturedReq.Header.Get("X-Custom-Header"))
	assert.Equal(t, "application/json", rt.capturedReq.Header.Get("Content-Type"))
}

func TestGCPCloudFunctionsTrigger_Execute_GetMethod(t *testing.T) {
	rt := &fakeRoundTripper{statusCode: 200}
	trigger := getGCPTrigger()
	trigger.Client = &http.Client{Transport: rt}
	trigger.Trigger.Template.GCPCloudFunctions.Method = http.MethodGet
	trigger.Trigger.Template.GCPCloudFunctions.Payload = nil

	resource, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)

	result, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, resource)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, http.MethodGet, rt.capturedReq.Method)
	assert.Empty(t, rt.capturedReq.Header.Get("Content-Type"))
}
