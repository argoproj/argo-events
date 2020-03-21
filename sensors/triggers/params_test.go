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

package triggers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Details struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Pin    string `json:"pin"`
}

type Payload struct {
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Details   Details `json:"details"`
}

func TestConstructPayload(t *testing.T) {
	obj := sensorObj.DeepCopy()
	id := obj.NodeID("fake-dependency")
	id2 := obj.NodeID("another-fake-dependency")

	obj.Status = v1alpha1.SensorStatus{
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
					Data: []byte("{\"firstName\": \"fake\"}"),
				},
			},
			id2: {
				Name: "another-fake-dependency",
				Type: v1alpha1.NodeTypeEventDependency,
				ID:   id,
				Event: &apicommon.Event{
					Context: apicommon.EventContext{
						ID:              "2",
						Type:            "calendar",
						Source:          "calendar-gateway",
						DataContentType: "application/json",
						SpecVersion:     "0.3",
						Subject:         "example-1",
					},
					Data: []byte("{\"lastName\": \"foo\"}"),
				},
			},
		},
	}

	defaultFirstName := "faker"
	defaultLastName := "bar"

	parameters := []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
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
	}

	payloadBytes, err := ConstructPayload(obj, parameters)
	assert.Nil(t, err)
	assert.NotNil(t, payloadBytes)

	var p *Payload
	err = json.Unmarshal(payloadBytes, &p)
	assert.Nil(t, err)
	assert.Equal(t, "fake", p.FirstName)
	assert.Equal(t, "foo", p.LastName)

	parameters[0].Src.DataKey = "unknown"
	parameters[1].Src.DataKey = "unknown"

	payloadBytes, err = ConstructPayload(obj, parameters)
	assert.Nil(t, err)
	assert.NotNil(t, payloadBytes)

	err = json.Unmarshal(payloadBytes, &p)
	assert.Nil(t, err)
	assert.Equal(t, "faker", p.FirstName)
	assert.Equal(t, "bar", p.LastName)
}

func TestExtractEvents(t *testing.T) {
	obj := sensorObj.DeepCopy()
	id := obj.NodeID("fake-dependency")
	obj.Status = v1alpha1.SensorStatus{
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
					Data: []byte("{\"name\": \"fake\"}"),
				},
			},
		},
	}
	events := ExtractEvents(obj, []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "name",
			},
		},
	})
	assert.NotNil(t, events)
	assert.Equal(t, events["fake-dependency"].Context.Subject, "example-1")

	delete(obj.Status.Nodes, id)
	events = ExtractEvents(obj, []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "name",
			},
		},
	})
	assert.Empty(t, events)
}

func TestResolveParamValue(t *testing.T) {
	event := apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: "application/json",
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.MicroTime{Time: time.Now()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
	}
	eventBody, err := json.Marshal(event)
	assert.Nil(t, err)

	events := map[string]apicommon.Event{
		"fake-dependency": event,
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
				DependencyName: "fake-dependency",
				DataKey:        "name.first",
			},
			result: "fake",
		},
		{
			name: "get the event subject",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				ContextKey:     "subject",
			},
			result: "example-1",
		},
		{
			name: "get the entire payload",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
			},
			result: string(eventBody),
		},
		{
			name: "get the default value",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				Value:          &defaultValue,
			},
			result: defaultValue,
		},
		{
			name: "data key has preference over context key",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				ContextKey:     "subject",
				DataKey:        "name.first",
			},
			result: "fake",
		},
		{
			name: "get first name with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataTemplate:   "{{ .Input.name.first }}",
			},
			result: "fake",
		},
		{
			name: "get capitalized first name with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataTemplate:   "{{ upper .Input.name.first }}",
			},
			result: "FAKE",
		},
		{
			name: "get subject with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName:  "fake-dependency",
				ContextTemplate: "{{ .Input.subject }}",
			},
			result: "example-1",
		},
		{
			name: "get formatted subject with template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName:  "fake-dependency",
				ContextTemplate: `{{ .Input.subject | replace "-" "_" }}`,
			},
			result: "example_1",
		},
		{
			name: "data template has preference over context template",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName:  "fake-dependency",
				ContextTemplate: "{{ .Input.subject }}",
				DataTemplate:    "{{ .Input.name.first }}",
			},
			result: "fake",
		},
		{
			name: "data template fails over to data key",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataTemplate:   "{{ .Input.name.non_exist }}",
				DataKey:        "name.first",
			},
			result: "fake",
		},
		{
			name: "invalid template fails over to data key",
			source: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataTemplate:   "{{ no }}",
				DataKey:        "name.first",
			},
			result: "fake",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ResolveParamValue(test.source, events)
			assert.Nil(t, err)
			assert.Equal(t, test.result, string(result))
		})
	}
}

func TestRenderDataAsJSON(t *testing.T) {
	event := &apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: common.MediaTypeJSON,
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.MicroTime{Time: time.Now()},
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
	event.Context.DataContentType = common.MediaTypeYAML
	body, err = renderEventDataAsJSON(event)
	assert.Nil(t, err)
	assert.Equal(t, string(body), "{\"age\":20,\"name\":\"test\"}")
}

func TestApplyParams(t *testing.T) {
	event := apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: common.MediaTypeJSON,
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.MicroTime{Time: time.Now()},
		},
		Data: []byte("{\"name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
	}

	events := map[string]apicommon.Event{
		"fake-dependency": event,
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
						DependencyName: "fake-dependency",
						DataKey:        "name.first",
					},
					Dest:      "name",
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
						DependencyName: "fake-dependency",
						DataKey:        "name.first",
					},
					Dest:      "name",
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
						DependencyName: "fake-dependency",
						DataKey:        "name.first",
					},
					Dest:      "name",
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
						DependencyName: "fake-dependency",
						DataKey:        "name.first",
					},
					Dest:      "name",
					Operation: v1alpha1.TriggerParameterOpOverwrite,
				},
			},
			jsonObj: []byte("{\"name\": \"faker\"}"),
			result:  []byte("{\"name\": \"fake\"}"),
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
	deployment := newUnstructured("apps/v1", "Deployment", "fake-deployment", "fake")

	event := apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: common.MediaTypeJSON,
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.MicroTime{Time: time.Now()},
		},
		Data: []byte("{\"name\": {\"first\": \"test-deployment\"} }"),
	}

	obj.Spec.Triggers[0].Template.K8s.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}
	id := obj.NodeID("fake-dependency")
	obj.Status.Nodes = map[string]v1alpha1.NodeStatus{
		id: {
			Event: &event,
			ID:    id,
			Name:  "fake-dependency",
			Type:  v1alpha1.NodeTypeEventDependency,
		},
	}
	obj.Spec.Triggers[0].Template.K8s.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "name.first",
			},
			Operation: v1alpha1.TriggerParameterOpNone,
			Dest:      "metadata.name",
		},
	}

	err := ApplyResourceParameters(obj, obj.Spec.Triggers[0].Template.K8s.Parameters, deployment)
	assert.Nil(t, err)
	assert.Equal(t, deployment.GetName(), "test-deployment")
}

func TestApplyTemplateParameters(t *testing.T) {
	obj := sensorObj.DeepCopy()
	event := apicommon.Event{
		Context: apicommon.EventContext{
			DataContentType: common.MediaTypeJSON,
			Subject:         "example-1",
			SpecVersion:     "0.3",
			Source:          "webhook-gateway",
			Type:            "webhook",
			ID:              "1",
			Time:            metav1.MicroTime{Time: time.Now()},
		},
		Data: []byte("{\"group\": \"fake\" }"),
	}
	id := obj.NodeID("fake-dependency")
	obj.Status.Nodes = map[string]v1alpha1.NodeStatus{
		id: {
			Event: &event,
			ID:    id,
			Name:  "fake-dependency",
			Type:  v1alpha1.NodeTypeEventDependency,
		},
	}
	obj.Spec.Triggers[0].Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "group",
			},
			Operation: v1alpha1.TriggerParameterOpOverwrite,
			Dest:      "k8s.group",
		},
	}
	err := ApplyTemplateParameters(obj, &obj.Spec.Triggers[0])
	assert.Nil(t, err)
	assert.Equal(t, "fake", obj.Spec.Triggers[0].Template.K8s.GroupVersionResource.Group)
}
