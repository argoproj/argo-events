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

package resource

import (
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// ResourceEventSourceExecutor implements Eventing
type ResourceEventSourceExecutor struct {
	Log zerolog.Logger
	// K8RestConfig is kubernetes cluster config
	K8RestConfig *rest.Config
}

// resource refers to a dependency on a k8s resource.
// +k8s:openapi-gen=true
type resource struct {
	Namespace string          `json:"namespace"`
	Filter    *ResourceFilter `json:"filter,omitempty"`
	Version   string `json:"version"`
	// +k8s:openapi-gen=false
	metav1.GroupVersionKind `json:",inline"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource signal objects
// +k8s:openapi-gen=true
type ResourceFilter struct {
	Prefix      string            `json:"prefix,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	// +k8s:openapi-gen=false
	CreatedBy metav1.Time `json:"createdBy,omitempty"`
}

func parseEventSource(es *string) (*resource, error) {
	var r *resource
	err := yaml.Unmarshal([]byte(*es), &r)
	if err != nil {
		return nil, err
	}
	return r, err
}
