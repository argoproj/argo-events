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
	"k8s.io/client-go/util/workqueue"
)

// ResourceEventSourceExecutor implements Eventing
type ResourceEventSourceExecutor struct {
	Log zerolog.Logger
	// K8RestConfig is kubernetes cluster config
	K8RestConfig *rest.Config
}

// resource refers to a dependency on a k8s resource.
type resource struct {
	// Namespace where resource is deployed
	Namespace string `json:"namespace"`
	// Filter is applied on the metadata of the resource
	Filter *ResourceFilter `json:"filter,omitempty"`
	// Version of the source
	Version string `json:"version"`
	// Group of the resource
	metav1.GroupVersionResource `json:",inline"`
	//
	queue workqueue.RateLimitingInterface
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
	r.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	return r, err
}
