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
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

const ArgoEventsEventSourceVersion = "v0.10"

// ResourceEventSourceExecutor implements Eventing
type ResourceEventSourceExecutor struct {
	Log *logrus.Logger
	// K8RestConfig is kubernetes cluster config
	K8RestConfig *rest.Config
}

// resource refers to a dependency on a k8s resource.
type resource struct {
	// Namespace where resource is deployed
	Namespace string `json:"namespace"`
	// Filter is applied on the metadata of the resource
	Filter *ResourceFilter `json:"filter,omitempty"`
	// Group of the resource
	metav1.GroupVersionKind `json:",inline"`
	// Type is the event type. Refer https://github.com/kubernetes/apimachinery/blob/dcb391cde5ca0298013d43336817d20b74650702/pkg/watch/watch.go#L43
	// If not provided, the gateway will watch all events for a resource.
	Type watch.EventType `json:"type,omitempty"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource event objects
type ResourceFilter struct {
	Prefix      string            `json:"prefix,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedBy   metav1.Time       `json:"createdBy,omitempty"`
}

func parseEventSource(es string) (interface{}, error) {
	var r *resource
	err := yaml.Unmarshal([]byte(es), &r)
	if err != nil {
		return nil, err
	}
	return r, err
}
