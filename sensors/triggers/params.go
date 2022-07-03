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
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// ConstructPayload constructs a payload for operations involving request and responses like HTTP request.
func ConstructPayload(events map[string]*v1alpha1.Event, parameters []v1alpha1.TriggerParameter) ([]byte, error) {
	var payload []byte

	for _, parameter := range parameters {
		value, err := ResolveParamValue(parameter.Src, events)
		if err != nil {
			return nil, err
		}
		tmp, err := sjson.SetBytes(payload, parameter.Dest, *value)
		if err != nil {
			return nil, err
		}
		payload = tmp
	}

	return payload, nil
}

// ApplyTemplateParameters applies parameters to trigger template
func ApplyTemplateParameters(events map[string]*v1alpha1.Event, trigger *v1alpha1.Trigger) error {
	if trigger.Parameters != nil && len(trigger.Parameters) > 0 {
		templateBytes, err := json.Marshal(trigger.Template)
		if err != nil {
			return err
		}
		tObj, err := ApplyParams(templateBytes, trigger.Parameters, events)
		if err != nil {
			return err
		}
		tmplt := &v1alpha1.TriggerTemplate{}
		if err = json.Unmarshal(tObj, tmplt); err != nil {
			return err
		}
		trigger.Template = tmplt
	}
	return nil
}

// ApplyResourceParameters applies parameters to K8s resource within trigger
func ApplyResourceParameters(events map[string]*v1alpha1.Event, parameters []v1alpha1.TriggerParameter, obj *unstructured.Unstructured) error {
	if parameters != nil {
		jObj, err := obj.MarshalJSON()
		if err != nil {
			return err
		}
		jUpdatedObj, err := ApplyParams(jObj, parameters, events)
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
func ApplyParams(jsonObj []byte, params []v1alpha1.TriggerParameter, events map[string]*v1alpha1.Event) ([]byte, error) {
	for _, param := range params {
		// let's grab the param value
		value, err := ResolveParamValue(param.Src, events)
		if err != nil {
			return nil, err
		}
		if value == nil {
			continue
		}

		switch op := param.Operation; op {
		case v1alpha1.TriggerParameterOpAppend, v1alpha1.TriggerParameterOpPrepend:
			// prepend or append the current value
			current := gjson.GetBytes(jsonObj, param.Dest)

			if current.Exists() {
				if op == v1alpha1.TriggerParameterOpAppend {
					*value = current.String() + *value
				} else {
					*value += current.String()
				}
			}
		case v1alpha1.TriggerParameterOpOverwrite, v1alpha1.TriggerParameterOpNone:
			// simply overwrite the current value with the new one
		default:
			return nil, fmt.Errorf("unsupported trigger parameter operation: %+v", op)
		}

		// now let's set the value
		tmp, err := sjson.SetBytes(jsonObj, param.Dest, *value)
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
func renderEventDataAsJSON(event *v1alpha1.Event) ([]byte, error) {
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
func ResolveParamValue(src *v1alpha1.TriggerParameterSource, events map[string]*v1alpha1.Event) (*string, error) {
	var err error
	var eventPayload []byte
	var key string
	var tmplt string
	var resultValue string

	event, eventExists := events[src.DependencyName]
	switch {
	case eventExists:
		// If context or data keys are not set, return the event payload as is
		if src.ContextKey == "" && src.DataKey == "" && src.DataTemplate == "" && src.ContextTemplate == "" {
			eventPayload, err = json.Marshal(&event)
		}
		// Get the context bytes
		if src.ContextKey != "" || src.ContextTemplate != "" {
			key = src.ContextKey
			tmplt = src.ContextTemplate
			eventPayload, err = json.Marshal(&event.Context)
		}
		// Get the payload bytes
		if src.DataKey != "" || src.DataTemplate != "" {
			key = src.DataKey
			tmplt = src.DataTemplate
			eventPayload, err = renderEventDataAsJSON(event)
		}
	case src.Value != nil:
		// Use the default value set by the user in case the event is missing
		resultValue = *src.Value
		return &resultValue, nil
	default:
		// The parameter doesn't have a default value and is referencing a dependency that is
		// missing in the received events. This is not an error and may happen with || conditions.
		return nil, nil
	}

	if err != nil {
		if src.Value != nil {
			fmt.Printf("failed to parse the event data, using default value. err: %+v\n", err)
			resultValue = *src.Value
			return &resultValue, nil
		}
		return nil, err
	}
	// Get the value corresponding to specified key within JSON object
	if eventPayload != nil {
		if tmplt != "" {
			resultValue, err = getValueWithTemplate(eventPayload, tmplt)
			if err == nil {
				return &resultValue, nil
			}
			fmt.Printf("failed to execute the src event template, falling back to key or value. err: %+v\n", err)
		}
		if key != "" {
			resultValue, err = getValueByKey(eventPayload, key)
			if err == nil {
				return &resultValue, nil
			}
			fmt.Printf("Failed to get value by key: %+v\n", err)
		}
		if src.Value != nil {
			resultValue = *src.Value
			return &resultValue, nil
		}

		resultValue = string(eventPayload)
		return &resultValue, nil
	}

	return nil, fmt.Errorf("unable to resolve '%s' parameter value", src.DependencyName)
}

// getValueWithTemplate will attempt to execute the provided template against
// the raw json bytes and then returns the result or any error
func getValueWithTemplate(value []byte, templString string) (string, error) {
	res := gjson.ParseBytes(value)
	tpl, err := template.New("param").Funcs(sprig.HermeticTxtFuncMap()).Parse(templString)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, map[string]interface{}{
		"Input": res.Value(),
	}); err != nil {
		return "", err
	}
	out := buf.String()
	if out == "" || out == "<no value>" {
		return "", fmt.Errorf("template evaluated to empty string or no value: %s", templString)
	}
	return out, nil
}

// getValueByKey will return the value in the raw json bytes at the provided key,
// or an error if it does not exist.
func getValueByKey(value []byte, key string) (string, error) {
	res := gjson.GetBytes(value, key)
	if res.Exists() {
		return res.String(), nil
	}
	return "", fmt.Errorf("key %s does not exist to in the event object\n", key)
}
