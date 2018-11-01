package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Resource refers to a dependency on a k8s resource.
// +k8s:openapi-gen=true
type resource struct {
	Namespace               string          `json:"namespace"`
	Filter                  *ResourceFilter `json:"filter,omitempty"`
	metav1.GroupVersionKind `json:",inline"`
}

// ResourceFilter contains K8 ObjectMeta information to further filter resource signal objects
type ResourceFilter struct {
	Prefix      string            `json:"prefix,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedBy   metav1.Time       `json:"createdBy,omitempty"`
}
