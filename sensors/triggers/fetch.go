/*
Copyright 2020 BlackRock, Inc.

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
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/artifacts"
)

func FetchKubernetesResource(source *v1alpha1.ArtifactLocation) (*unstructured.Unstructured, error) {
	if source == nil {
		return nil, fmt.Errorf("trigger source for k8s is empty")
	}
	creds, err := artifacts.GetCredentials(source)
	if err != nil {
		return nil, err
	}
	reader, err := artifacts.GetArtifactReader(source, creds)
	if err != nil {
		return nil, err
	}

	// uObj will either hold the resource definition stored in the trigger or just
	// a stub to provide enough information to fetch the object from K8s cluster
	uObj, err := artifacts.FetchArtifact(reader)
	if err != nil {
		return nil, err
	}
	return uObj, nil
}
