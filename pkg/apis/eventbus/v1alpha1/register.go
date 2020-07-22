// Package v1alpha1 contains API Schema definitions for the sources v1alpha1 API group
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen=package,register
// +k8s:defaulter-gen=TypeMeta
// +groupName=argoproj.io
package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/argoproj/argo-events/pkg/apis/eventbus"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: eventbus.Group, Version: "v1alpha1"}

	// SchemaGroupVersionKind is a group version kind used to attach owner references
	SchemaGroupVersionKind = schema.GroupVersionKind{Group: eventbus.Group, Version: "v1alpha1", Kind: eventbus.Kind}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme is required by pkg/client/...
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&EventBus{},
		&EventBusList{},
	)
	v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
