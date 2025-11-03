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
package http

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
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
					HTTP: &v1alpha1.HTTPTrigger{
						URL:     "http://fake.com:12000",
						Method:  "POST",
						Timeout: 10,
					},
				},
			},
		},
	},
}

var sensorObjHost = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor-host",
		Namespace: "fake",
	},
	Spec: v1alpha1.SensorSpec{
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger-host",
					HTTP: &v1alpha1.HTTPTrigger{
						URL:     "http://fake.com:12000",
						Method:  "POST",
						Timeout: 10,
						Host:    "fake-host.com",
					},
				},
			},
		},
	},
}

func getFakeHTTPTrigger() *HTTPTrigger {
	return &HTTPTrigger{
		Client:  nil,
		Sensor:  sensorObj.DeepCopy(),
		Trigger: sensorObj.Spec.Triggers[0].DeepCopy(),
		Logger:  logging.NewArgoEventsLogger(),
	}
}

func TestHTTPTrigger_FetchResource(t *testing.T) {
	trigger := getFakeHTTPTrigger()
	obj, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, obj)
	trigger1, ok := obj.(*v1alpha1.HTTPTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, trigger.Trigger.Template.HTTP.URL, trigger1.URL)
}

func getFakeHTTPTriggerHost() *HTTPTrigger {
	return &HTTPTrigger{
		Client:  nil,
		Sensor:  sensorObjHost.DeepCopy(),
		Trigger: sensorObjHost.Spec.Triggers[0].DeepCopy(),
		Logger:  logging.NewArgoEventsLogger(),
	}
}

func TestHTTPTrigger_FetchResourceHost(t *testing.T) {
	trigger := getFakeHTTPTriggerHost()
	obj, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, obj)
	trigger1, ok := obj.(*v1alpha1.HTTPTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, trigger.Trigger.Template.HTTP.URL, trigger1.URL)
	assert.Equal(t, trigger.Trigger.Template.HTTP.Host, trigger1.Host)
}

func TestHTTPTrigger_ApplyResourceParameters(t *testing.T) {
	trigger := getFakeHTTPTrigger()

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
			Data: []byte(`{"url": "http://another-fake.com", "method": "GET"}`),
		},
	}

	defaultValue := "http://default.com"
	secureHeader := &v1alpha1.SecureHeader{Name: "test", ValueFrom: &v1alpha1.ValueFromSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "tokens",
			},
			Key: "serviceToken"}},
	}

	defaultHeader := "application/json"
	secureHeaders := []*v1alpha1.SecureHeader{}
	secureHeaders = append(secureHeaders, secureHeader)
	trigger.Trigger.Template.HTTP.SecureHeaders = secureHeaders
	trigger.Trigger.Template.HTTP.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "url",
				Value:          &defaultValue,
			},
			Dest: "serverURL",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "method",
				Value:          &defaultValue,
			},
			Dest: "method",
		},
	}
	trigger.Trigger.Template.HTTP.DynamicHeaders = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "Content-Type",
				Value:          &defaultHeader,
			},
			Dest: "X-Content-Type",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "X-Github-Event",
				Value:          &defaultHeader,
			},
			Dest: "X-Github-Event",
		},
	}

	assert.Equal(t, http.MethodPost, trigger.Trigger.Template.HTTP.Method)

	resource, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.HTTP)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	updatedTrigger, ok := resource.(*v1alpha1.HTTPTrigger)
	assert.Equal(t, "serviceToken", updatedTrigger.SecureHeaders[0].ValueFrom.SecretKeyRef.Key)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "http://fake.com:12000", updatedTrigger.URL)
	assert.Equal(t, http.MethodGet, updatedTrigger.Method)
}

func TestHTTPTrigger_ApplyPolicy(t *testing.T) {
	trigger := getFakeHTTPTrigger()
	trigger.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{200, 300}},
	}
	response := &http.Response{StatusCode: 200}
	err := trigger.ApplyPolicy(context.TODO(), response)
	assert.Nil(t, err)

	trigger.Trigger.Policy = &v1alpha1.TriggerPolicy{
		Status: &v1alpha1.StatusPolicy{Allow: []int32{300}},
	}
	err = trigger.ApplyPolicy(context.TODO(), response)
	assert.NotNil(t, err)
}

// roundTripperMarker implements http.RoundTripper and marks called if invoked
type roundTripperMarker struct{ called bool }

func (r *roundTripperMarker) RoundTrip(req *http.Request) (*http.Response, error) {
	r.called = true
	return nil, fmt.Errorf("RoundTrip should not be called")
}

func TestHTTPTrigger_Execute_BasicAuthPasswordError(t *testing.T) {
	transport := &roundTripperMarker{}
	client := &http.Client{
		Transport: transport,
	}
	logger := logging.NewArgoEventsLogger()

	trig := &v1alpha1.HTTPTrigger{
		URL:    "http://example.invalid",
		Method: http.MethodGet,
		BasicAuth: &v1alpha1.BasicAuth{
			// Intentionally reference a non-existent secret to force GetSecretFromVolume error
			Password: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "missing"},
				Key:                  "pwd",
			},
		},
	}

	trigger := &HTTPTrigger{
		Client:  client,
		Sensor:  sensorObj.DeepCopy(),
		Trigger: (&v1alpha1.Trigger{Template: &v1alpha1.TriggerTemplate{Name: "basic-auth", HTTP: trig}}).DeepCopy(),
		Logger:  logger,
	}

	res, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trig)
	assert.Nil(t, res)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve the password")
	// Ensure the HTTP client was not used
	assert.False(t, transport.called)
}
