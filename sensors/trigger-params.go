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

package sensors

import (
	"fmt"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// apply the params to the resource json object
func applyParams(jsonObj []byte, params []v1alpha1.TriggerParameter, events map[string]apicommon.Event) ([]byte, error) {
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

// helper method to resolve the parameter's value from the src
// returns an error if the Path is invalid/not found and the default value is nil OR if the eventDependency event doesn't exist and default value is nil
func resolveParamValue(src *v1alpha1.TriggerParameterSource, events map[string]apicommon.Event) (string, error) {
	if e, ok := events[src.Event]; ok {
		// only convert payload to json when path is set.
		if src.Path == "" {
			return string(e.Payload), nil
		}
		js, err := renderEventDataAsJSON(&e)
		if err != nil {
			fmt.Printf("failed to render event data as json. err: %+v", err)
			if src.Value != nil {
				return *src.Value, nil
			}
			return "", fmt.Errorf("failed to render event payload as JSON. err:%+v", err)
		}
		res := gjson.GetBytes(js, src.Path)
		if res.Exists() {
			return res.String(), nil
		}
	}
	if src.Value != nil {
		return *src.Value, nil
	}
	return "", fmt.Errorf("unable to resolve '%s' parameter value. verify the path: '%s' is valid and/or set a default value for this param", src.Event, src.Path)
}
