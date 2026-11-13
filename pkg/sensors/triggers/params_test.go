/*
Copyright 2018 The Argoproj Authors.

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
	"encoding/json"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

var sensorObj = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: testFake,
	},
	Spec: v1alpha1.SensorSpec{
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					K8s:  &v1alpha1.StandardK8STrigger{},
				},
			},
		},
	},
}

func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				testKeyName: name,
				"labels": map[string]interface{}{
					testKeyName: name,
				},
			},
		},
	}
}

type Details struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Pin    string `json:"pin"`
}

type Payload struct {
	FirstName        string  `json:"firstName"`
	LastName         string  `json:"lastName"`
	Age              int     `json:"age"`
	IsActive         bool    `json:"isActive"`
	TypelessAge      string  `json:"typelessAge"`
	TypelessIsActive string  `json:"typelessIsActive"`
	Details          Details `json:"details"`
}

func TestConstructPayload(t *testing.T) {
	testEvents := map[string]*v1alpha1.Event{
		testDependency: {
			Context: &v1alpha1.EventContext{
				ID:              "1",
				Type:            testWebhook,
				Source:          testSource,
				DataContentType: v1alpha1.MediaTypeJSON,
				Subject:         testExample1,
			},
			Data: []byte("{\"firstName\": \"fake\"}"),
		},
		"another-fake-dependency": {
			Context: &v1alpha1.EventContext{
				ID:              "2",
				Type:            "calendar",
				Source:          "calendar-gateway",
				DataContentType: v1alpha1.MediaTypeJSON,
				Subject:         testExample1,
			},
			Data: []byte("{\"lastName\": \"foo\"}"),
		},
		testUseEventDataType: {
			Context: &v1alpha1.EventContext{
				ID:              "3",
				Type:            "calendar",
				Source:          "calendar-gateway",
				DataContentType: v1alpha1.MediaTypeJSON,
				Subject:         testExample1,
			},
			Data: []byte("{\"age\": 100, \"isActive\": false, \"countries\": [\"ca\", \"us\", \"mx\"]}"),
		},
	}

	defaultFirstName := "faker"
	defaultLastName := "bar"

	parameters := []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataKey:        "firstName",
				Value:          &defaultFirstName,
			},
			Dest: "firstName",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "another-fake-dependency",
				DataKey:        "lastName",
				Value:          &defaultLastName,
			},
			Dest: "lastName",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testUseEventDataType,
				DataKey:        testKeyAge,
				UseRawData:     true,
			},
			Dest: testKeyAge,
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testUseEventDataType,
				DataKey:        testKeyIsActive,
				UseRawData:     true,
			},
			Dest: testKeyIsActive,
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testUseEventDataType,
				DataKey:        testKeyAge,
			},
			Dest: "typelessAge",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testUseEventDataType,
				DataKey:        testKeyIsActive,
			},
			Dest: "typelessIsActive",
		},
	}

	payloadBytes, err := ConstructPayload(testEvents, parameters)
	assert.Nil(t, err)
	assert.NotNil(t, payloadBytes)

	var p *Payload
	err = json.Unmarshal(payloadBytes, &p)
	assert.Nil(t, err)
	assert.Equal(t, testFake, p.FirstName)
	assert.Equal(t, "foo", p.LastName)
	assert.Equal(t, 100, p.Age)
	assert.Equal(t, false, p.IsActive)
	assert.Equal(t, "100", p.TypelessAge)
	assert.Equal(t, "false", p.TypelessIsActive)

	parameters[0].Src.DataKey = "unknown"
	parameters[1].Src.DataKey = "unknown"

	payloadBytes, err = ConstructPayload(testEvents, parameters)
	assert.Nil(t, err)
	assert.NotNil(t, payloadBytes)

	err = json.Unmarshal(payloadBytes, &p)
	assert.Nil(t, err)
	assert.Equal(t, "faker", p.FirstName)
	assert.Equal(t, "bar", p.LastName)
}

func TestResolveParamValue(t *testing.T) {
	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			DataContentType: v1alpha1.MediaTypeJSON,
			Subject:         testExample1,
			Source:          testSource,
			Type:            testWebhook,
			ID:              "1",
			Time:            metav1.Time{Time: time.Now().UTC()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"}, \"reviews\": 8, \"rating\": 4.5, \"isActive\" : true, \"isVerified\" : false, \"countries\": [\"ca\", \"us\", \"mx\"]}"),
	}
	eventBody, err := json.Marshal(event)
	assert.Nil(t, err)

	events := map[string]*v1alpha1.Event{
		testDependency: event,
	}

	defaultValue := "hello"

	tests := []struct {
		name   string
		source *v1alpha1.TriggerParameterSource
		result string
	}{
		{
			name: "get first name",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataKey:        testKeyNameFirst,
			},
			result: testFake,
		},
		{
			name: "get the event subject",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				ContextKey:     "subject",
			},
			result: testExample1,
		},
		{
			name: "get the entire payload",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
			},
			result: string(eventBody),
		},
		{
			name: "get the default value",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				Value:          &defaultValue,
			},
			result: defaultValue,
		},
		{
			name: "data key has preference over context key",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				ContextKey:     "subject",
				DataKey:        testKeyNameFirst,
			},
			result: testFake,
		},
		{
			name: "get first name with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataTemplate:   "{{ .Input.name.first }}",
			},
			result: testFake,
		},
		{
			name: "get capitalized first name with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataTemplate:   "{{ upper .Input.name.first }}",
			},
			result: "FAKE",
		},
		{
			name: "get subject with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName:  testDependency,
				ContextTemplate: "{{ .Input.subject }}",
			},
			result: testExample1,
		},
		{
			name: "get formatted subject with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName:  testDependency,
				ContextTemplate: `{{ .Input.subject | replace "-" "_" }}`,
			},
			result: "example_1",
		},
		{
			name: "data template has preference over context template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName:  testDependency,
				ContextTemplate: "{{ .Input.subject }}",
				DataTemplate:    "{{ .Input.name.first }}",
			},
			result: testFake,
		},
		{
			name: "data template fails over to data key",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataTemplate:   "{{ .Input.name.non_exist }}",
				DataKey:        testKeyNameFirst,
			},
			result: testFake,
		},
		{
			name: "invalid template fails over to data key",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataTemplate:   "{{ no }}",
				DataKey:        testKeyNameFirst,
			},
			result: testFake,
		},
		{
			name: "UseRawData set to true - string",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataKey:        testKeyNameFirst,
				UseRawData:     true,
			},
			result: testFake,
		},
		{
			name: "UseRawData set to true - json",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataKey:        testKeyName,
				UseRawData:     true,
			},
			result: "{\"first\": \"fake\", \"last\": \"user\"}",
		},
		{
			name: "UseRawData set to true - list",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataKey:        testKeyCountries,
				UseRawData:     true,
			},
			result: "[\"ca\", \"us\", \"mx\"]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, _, err := ResolveParamValue(test.source, events)
			assert.Nil(t, err)
			assert.Equal(t, test.result, *result)
		})
	}
}

func TestRenderDataAsJSON(t *testing.T) {
	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			DataContentType: v1alpha1.MediaTypeJSON,
			Subject:         testExample1,
			Source:          testSource,
			Type:            testWebhook,
			ID:              "1",
			Time:            metav1.Time{Time: time.Now().UTC()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
	}
	body, err := renderEventDataAsJSON(event)
	assert.Nil(t, err)
	assert.Equal(t, string(body), "{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }")

	testYaml := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{
		Name: "test",
		Age:  20,
	}

	yamlBody, err := yaml.Marshal(&testYaml)
	assert.Nil(t, err)
	event.Data = yamlBody
	event.Context.DataContentType = v1alpha1.MediaTypeYAML
	assert.Nil(t, err)
	body, err = renderEventDataAsJSON(event)
	assert.Nil(t, err)
	assert.Equal(t, string(body), "{\"age\":20,\"name\":\"test\"}")
}

func TestApplyParams(t *testing.T) {
	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			DataContentType: v1alpha1.MediaTypeJSON,
			Subject:         testExample1,
			Source:          testSource,
			Type:            testWebhook,
			ID:              "1",
			Time:            metav1.Time{Time: time.Now().UTC()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"}, \"age\": 100, \"countries\": [\"ca\", \"us\", \"mx\"] }"),
	}

	events := map[string]*v1alpha1.Event{
		testDependency: event,
	}

	tests := []struct {
		name    string
		params  []v1alpha1.TriggerParameter
		jsonObj []byte
		result  []byte
	}{
		{
			name: "normal apply parameters operation",
			params: []v1alpha1.TriggerParameter{
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyNameFirst,
					},
					Dest:      testKeyName,
					Operation: v1alpha1.TriggerParameterOpNone,
				},
			},
			jsonObj: []byte("{\"name\": \"faker\"}"),
			result:  []byte("{\"name\": \"fake\"}"),
		},
		{
			name: "apply parameters with prepend operation",
			params: []v1alpha1.TriggerParameter{
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyNameFirst,
					},
					Dest:      testKeyName,
					Operation: v1alpha1.TriggerParameterOpPrepend,
				},
			},
			jsonObj: []byte("{\"name\": \"faker\"}"),
			result:  []byte("{\"name\": \"fakefaker\"}"),
		},
		{
			name: "apply parameters with append operation",
			params: []v1alpha1.TriggerParameter{
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyNameFirst,
					},
					Dest:      testKeyName,
					Operation: v1alpha1.TriggerParameterOpAppend,
				},
			},
			jsonObj: []byte("{\"name\": \"faker\"}"),
			result:  []byte("{\"name\": \"fakerfake\"}"),
		},
		{
			name: "apply parameters with overwrite operation",
			params: []v1alpha1.TriggerParameter{
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyNameFirst,
					},
					Dest:      testKeyName,
					Operation: v1alpha1.TriggerParameterOpOverwrite,
				},
			},
			jsonObj: []byte("{\"name\": \"faker\"}"),
			result:  []byte("{\"name\": \"fake\"}"),
		},
		{
			name: "apply block parameters with overwrite operation - useRawDataValue false",
			params: []v1alpha1.TriggerParameter{
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyName,
					},
					Dest:      testKeyName,
					Operation: v1alpha1.TriggerParameterOpOverwrite,
				},
			},
			jsonObj: []byte("{\"name\": \"faker\"}"),
			result:  []byte("{\"name\": \"{\\\"first\\\": \\\"fake\\\", \\\"last\\\": \\\"user\\\"}\"}"),
		},
		{
			name: "apply block parameters with overwrite operation - useRawDataValue true",
			params: []v1alpha1.TriggerParameter{
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyName,
						UseRawData:     true,
					},
					Dest:      testKeyName,
					Operation: v1alpha1.TriggerParameterOpOverwrite,
				},
			},
			jsonObj: []byte("{\"name\": \"faker\"}"),
			result:  []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"}}"),
		},
		{
			name: "Use raw data types",
			params: []v1alpha1.TriggerParameter{
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyAge,
						UseRawData:     true,
					},
					Dest:      testKeyAge,
					Operation: v1alpha1.TriggerParameterOpOverwrite,
				},
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyAge,
						UseRawData:     true,
					},
					Dest:      "ageWithYears",
					Operation: v1alpha1.TriggerParameterOpAppend,
				},
				{
					Src: &v1alpha1.TriggerParameterSource{
						DependencyName: testDependency,
						DataKey:        testKeyCountries,
						UseRawData:     true,
					},
					Dest:      testKeyCountries,
					Operation: v1alpha1.TriggerParameterOpAppend,
				},
			},
			jsonObj: []byte("{\"age\": \"this-gets-over-written\", \"ageWithYears\": \"Years: \"}"),
			result:  []byte("{\"age\": 100, \"ageWithYears\": \"Years: 100\",\"countries\":[\"ca\", \"us\", \"mx\"]}"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ApplyParams(test.jsonObj, test.params, events)
			assert.Nil(t, err)
			assert.Equal(t, string(test.result), string(result))
		})
	}
}

func TestApplyResourceParameters(t *testing.T) {
	obj := sensorObj.DeepCopy()
	deployment := newUnstructured("apps/v1", "Deployment", "fake-deployment", testFake)

	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			DataContentType: v1alpha1.MediaTypeJSON,
			Subject:         testExample1,
			Source:          testSource,
			Type:            testWebhook,
			ID:              "1",
			Time:            metav1.Time{Time: time.Now().UTC()},
		},
		Data: []byte("{\"name\": {\"first\": \"test-deployment\"} }"),
	}

	testEvents := map[string]*v1alpha1.Event{
		testDependency: event,
	}

	artifact := v1alpha1.NewK8SResource(deployment)
	obj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: &artifact,
	}
	obj.Spec.Triggers[0].Template.K8s.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataKey:        testKeyNameFirst,
			},
			Operation: v1alpha1.TriggerParameterOpNone,
			Dest:      "metadata.name",
		},
	}

	err := ApplyResourceParameters(testEvents, obj.Spec.Triggers[0].Template.K8s.Parameters, deployment)
	assert.Nil(t, err)
	assert.Equal(t, deployment.GetName(), "test-deployment")
}

func TestApplyTemplateParameters(t *testing.T) {
	obj := sensorObj.DeepCopy()
	event := &v1alpha1.Event{
		Context: &v1alpha1.EventContext{
			DataContentType: v1alpha1.MediaTypeJSON,
			Subject:         testExample1,
			Source:          testSource,
			Type:            testWebhook,
			ID:              "1",
			Time:            metav1.Time{Time: time.Now().UTC()},
		},
		Data: []byte("{\"group\": \"fake\" }"),
	}
	testEvents := map[string]*v1alpha1.Event{
		testDependency: event,
	}

	obj.Spec.Triggers[0].Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testDependency,
				DataKey:        "group",
			},
			Operation: v1alpha1.TriggerParameterOpOverwrite,
			Dest:      "k8s.group",
		},
	}
	err := ApplyTemplateParameters(testEvents, &obj.Spec.Triggers[0])
	assert.Nil(t, err)
}
func TestGetValueByKey_NullValue(t *testing.T) {
	jsonData := []byte(`{"nullable_key": null, "string_key": "value"}`)

	// Test null value
	result, typ, err := getValueByKey(jsonData, "nullable_key")
	assert.NoError(t, err)
	assert.Equal(t, "null", result)
	assert.Equal(t, "Null", typ)

	// Test string value for comparison
	result, typ, err = getValueByKey(jsonData, "string_key")
	assert.NoError(t, err)
	assert.Equal(t, "value", result)
	assert.Equal(t, "String", typ)
}

func TestConstructPayload_NullValue(t *testing.T) {
	events := map[string]*v1alpha1.Event{
		testDep: {
			Data: []byte(`{"nullable_key": null, "other_key": "value"}`),
			Context: &v1alpha1.EventContext{
				DataContentType: v1alpha1.MediaTypeJSON,
			},
		},
	}

	parameters := []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testDep,
				DataKey:        "nullable_key",
				UseRawData:     true,
			},
			Dest: "nullable_key",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: testDep,
				DataKey:        "other_key",
			},
			Dest: "other_key",
		},
	}

	payload, err := ConstructPayload(events, parameters)
	assert.NoError(t, err)

	// Verify the payload is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(payload, &result)
	assert.NoError(t, err)

	// Verify null value is preserved
	assert.Nil(t, result["nullable_key"])
	assert.Equal(t, "value", result["other_key"])

	// Verify the raw JSON contains "null" not empty
	assert.Contains(t, string(payload), `"nullable_key":null`)
}
