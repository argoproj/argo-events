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

package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
)

// various supported media types
// TODO: add support for XML
const (
	MediaTypeJSON string = "application/json"
	//MediaTypeXML  string = "application/xml"
	MediaTypeYAML string = "application/yaml"
)

// getNodeByName returns a copy of the node from this sensor for the nodename
// for signals this nodename should be the name of the signal
func (soc *sOperationCtx) getNodeByName(nodename string) *v1alpha1.NodeStatus {
	nodeID := soc.s.NodeID(nodename)
	node, ok := soc.s.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return node.DeepCopy()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// util method to render an event's data as a JSON []byte
// json is a subset of yaml so this should work...
func renderEventDataAsJSON(e *v1alpha1.Event) ([]byte, error) {
	if e == nil {
		return nil, fmt.Errorf("event is nil")
	}
	raw := e.Data
	// contentType is formatted as: '{type}; charset="xxx"'
	contents := strings.Split(e.Context.ContentType, ";")
	switch contents[0] {
	case MediaTypeJSON:
		if isJSON(raw) {
			return raw, nil
		}
		return nil, fmt.Errorf("event data is not valid JSON")
	case MediaTypeYAML:
		data, err := yaml.YAMLToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("failed converting yaml event data to JSON: %s", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported event content type: %s", e.Context.ContentType)
	}
}

func isJSON(b []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(b, &js) == nil
}
