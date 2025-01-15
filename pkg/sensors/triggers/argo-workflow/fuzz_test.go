/*
Copyright 2025 The Argoproj Authors.
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

package argo_workflow

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func bytesToUnstructuredFuzz(jsonBytes []byte) (*unstructured.Unstructured, error) {
	obj := make(map[string]interface{})
	err := yaml.Unmarshal(jsonBytes, &obj)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}
func FuzzArgoWorkflowTriggerExecute(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		ctx := context.Background()
		var actual string
		firstArg := "--foo"
		secondArg := "--bar"
		unstr, err := bytesToUnstructuredFuzz(data)
		if err != nil {
			return
		}
		trigger := storingCmdTrigger(&actual, firstArg, secondArg)
		_, err = namespacedClientFrom(trigger).Namespace(unstr.GetNamespace()).Create(ctx, unstr, metav1.CreateOptions{})
		if err != nil {
			return
		}
		_, _ = trigger.Execute(ctx, nil, unstr)
	})
}
