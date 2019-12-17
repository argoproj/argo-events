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
					Data: []byte("{\"Name\": \"fake\"}"),
				},
			},
		},
	}
	events := extractEvents(obj, []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				Event:   "fake-dependency",
				DataKey: "Name",
			},
		},
	})
	assert.NotNil(t, events)
	assert.Equal(t, events["fake-dependency"].Context.Subject, "example-1")

	delete(obj.Status.Nodes, id)
	events = extractEvents(obj, []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				Event:   "fake-dependency",
				DataKey: "Name",
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
		Data: []byte("{\"Name\": {\"first\": \"fake\", \"last\": \"user\"} }"),
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
			name: "get first Name",
			source: &v1alpha1.TriggerParameterSource{
				Event:   "fake-dependency",
				DataKey: "Name.first",
			},
			result: "fake",
		},
		{
			name: "get the event subject",
			source: &v1alpha1.TriggerParameterSource{
				Event:      "fake-dependency",
				ContextKey: "subject",
			},
			result: "example-1",
		},
		{
			name: "get the entire payload",
			source: &v1alpha1.TriggerParameterSource{
				Event: "fake-dependency",
			},
			result: string(eventBody),
		},
		{
			name: "get the default value",
			source: &v1alpha1.TriggerParameterSource{
				Event: "fake-dependency",
				Value: &defaultValue,
			},
			result: defaultValue,
		},
		{
			name: "data key has preference over context key",
			source: &v1alpha1.TriggerParameterSource{
				Event:      "fake-dependency",
				ContextKey: "subject",
				DataKey:    "Name.first",
			},
			result: "fake",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := resolveParamValue(test.source, events)
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
						Event:   "fake-dependency",
						DataKey: "name.first",
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
						Event:   "fake-dependency",
						DataKey: "name.first",
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
						Event:   "fake-dependency",
						DataKey: "name.first",
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
						Event:   "fake-dependency",
						DataKey: "name.first",
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
			result, err := applyParams(test.jsonObj, test.params, events)
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

	obj.Spec.Triggers[0].Template.Source = &v1alpha1.ArtifactLocation{
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
	obj.Spec.Triggers[0].ResourceParameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				Event:   "fake-dependency",
				DataKey: "name.first",
			},
			Operation: v1alpha1.TriggerParameterOpNone,
			Dest:      "metadata.name",
		},
	}

	err := ApplyResourceParameters(obj, obj.Spec.Triggers[0].ResourceParameters, deployment)
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
	obj.Spec.Triggers[0].TemplateParameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				Event:   "fake-dependency",
				DataKey: "group",
			},
			Operation: v1alpha1.TriggerParameterOpOverwrite,
			Dest:      "groupVersionResource.group",
		},
	}
	err := ApplyTemplateParameters(obj, &obj.Spec.Triggers[0])
	assert.Nil(t, err)
	assert.Equal(t, "fake", obj.Spec.Triggers[0].Template.GroupVersionResource.Group)
}
