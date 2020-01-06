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

package common

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	snctrl "github.com/argoproj/argo-events/controllers/sensor"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ApplyTemplateParameters applies parameters to trigger template
func ApplyTemplateParameters(sensor *v1alpha1.Sensor, trigger *v1alpha1.Trigger) error {
	if trigger.TemplateParameters != nil && len(trigger.TemplateParameters) > 0 {
		templateBytes, err := json.Marshal(trigger.Template)
		if err != nil {
			return err
		}
		tObj, err := ApplyParams(templateBytes, trigger.TemplateParameters, ExtractEvents(sensor, trigger.TemplateParameters))
		if err != nil {
			return err
		}
		template := &v1alpha1.TriggerTemplate{}
		if err = json.Unmarshal(tObj, template); err != nil {
			return err
		}
		trigger.Template = template
	}
	return nil
}

// ApplyResourceParameters applies parameters to K8s resource within trigger
func ApplyResourceParameters(sensor *v1alpha1.Sensor, parameters []v1alpha1.TriggerParameter, obj *unstructured.Unstructured) error {
	if parameters != nil && len(parameters) > 0 {
		jObj, err := obj.MarshalJSON()
		if err != nil {
			return err
		}
		jUpdatedObj, err := ApplyParams(jObj, parameters, ExtractEvents(sensor, parameters))
		if err != nil {
			return err
		}
		err = obj.UnmarshalJSON(jUpdatedObj)
		if err != nil {
			return err
		}
	}
	return nil
}

// ApplyParams applies the params to the resource json object
func ApplyParams(jsonObj []byte, params []v1alpha1.TriggerParameter, events map[string]apicommon.Event) ([]byte, error) {
	for _, param := range params {
		// let's grab the param value
		v, err := resolveParamValue(param.Src, events)
		if err != nil {
			return nil, err
		}

		switch op := param.Operation; op {
		case v1alpha1.TriggerParameterOpAppend, v1alpha1.TriggerParameterOpPrepend:
			// prepend or append the current value
			current := gjson.GetBytes(jsonObj, param.Dest)

			if current.Exists() {
				if op == v1alpha1.TriggerParameterOpAppend {
					v = current.String() + v
				} else {
					v = v + current.String()
				}
			}
		case v1alpha1.TriggerParameterOpOverwrite, v1alpha1.TriggerParameterOpNone:
			// simply overwrite the current value with the new one
		default:
			return nil, fmt.Errorf("unsupported trigger parameter operation: %+v", op)
		}

		// now let's set the value
		tmp, err := sjson.SetBytes(jsonObj, param.Dest, v)
		if err != nil {
			return nil, err
		}
		jsonObj = tmp
	}
	return jsonObj, nil
}

func isJSON(b []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(b, &js) == nil
}

// util method to render an event's data as a JSON []byte
// json is a subset of yaml so this should work...
func renderEventDataAsJSON(event *apicommon.Event) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}
	raw := event.Data
	switch event.Context.DataContentType {
	case common.MediaTypeJSON:
		if isJSON(raw) {
			return raw, nil
		}
		return nil, fmt.Errorf("event data is not valid JSON")
	case common.MediaTypeYAML:
		data, err := yaml.YAMLToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("failed converting yaml event data to JSON: %s", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported event content type: %s", event.Context.DataContentType)
	}
}

// helper method to resolve the parameter's value from the src
// returns an error if the Path is invalid/not found and the default value is nil OR if the eventDependency event doesn't exist and default value is nil
func resolveParamValue(src *v1alpha1.TriggerParameterSource, events map[string]apicommon.Event) (string, error) {
	var err error
	var value []byte
	var key string
	if event, ok := events[src.Event]; ok {
		// If context or data keys are not set, return the event payload as is
		if src.ContextKey == "" && src.DataKey == "" {
			value, err = json.Marshal(&event)
		}
		// Get the context bytes
		if src.ContextKey != "" {
			key = src.ContextKey
			value, err = json.Marshal(&event.Context)
		}
		// Get the payload bytes
		if src.DataKey != "" {
			key = src.DataKey
			value, err = renderEventDataAsJSON(&event)
		}
	}
	if err != nil && src.Value != nil {
		fmt.Printf("failed to parse the event data, using default value. err: %+v", err)
		return *src.Value, nil
	}
	// Get the value corresponding to specified key within JSON object
	if value != nil {
		if key != "" {
			res := gjson.GetBytes(value, key)
			if res.Exists() {
				return res.String(), nil
			}
			fmt.Printf("key %s does not exist to in the event object", key)
		}
		if src.Value != nil {
			return *src.Value, nil
		}
		return string(value), nil
	}
	return "", errors.Wrapf(err, "unable to resolve '%s' parameter value", src.Event)
}

// ExtractEvents is a helper method to extract the events from the event dependencies nodes associated with the resource params
// returns a map of the events keyed by the event dependency name
func ExtractEvents(sensor *v1alpha1.Sensor, params []v1alpha1.TriggerParameter) map[string]apicommon.Event {
	events := make(map[string]apicommon.Event)
	for _, param := range params {
		if param.Src != nil {
			node := snctrl.GetNodeByName(sensor, param.Src.Event)
			if node == nil {
				continue
			}
			if node.Event == nil {
				continue
			}
			events[param.Src.Event] = *node.Event
		}
	}
	return events
}
