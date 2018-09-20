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

package sensor

import (
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	v1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// apply the params to the resource json object
func applyParams(jsonObj []byte, params []v1alpha1.ResourceParameter, events map[string]v1alpha.Event) ([]byte, error) {
	tmp := make([]byte, len(jsonObj))
	for _, param := range params {
		// let's grab the param value
		v, err := resolveParamValue(param.Src, events)
		if err != nil {
			return nil, err
		}
		// now let's set the value
		tmp, err = sjson.SetBytes(jsonObj, param.Dest, v)
		if err != nil {
			return nil, err
		}
		jsonObj = tmp
	}
	return jsonObj, nil
}

// helper method to resolve the parameter's value from the src
// returns an error if the Path is invalid/not found and the default value is nil OR if the signal event doesn't exist and default value is nil
func resolveParamValue(src *v1alpha1.ResourceParameterSource, events map[string]v1alpha.Event) (string, error) {
	if e, ok := events[src.Signal]; ok {
		js, err := renderEventDataAsJSON(&e)
		if err != nil {
			if src.Value != nil {
				return *src.Value, nil
			}
			return "", err
		}
		res := gjson.GetBytes(js, src.Path)
		if res.Exists() {
			return res.String(), nil
		}
	}
	if src.Value != nil {
		return *src.Value, nil
	}
	return "", fmt.Errorf("unable to resolve '%s' parameter value. verify the path: '%s' is valid and/or set a default value for this param", src.Signal, src.Path)
}
