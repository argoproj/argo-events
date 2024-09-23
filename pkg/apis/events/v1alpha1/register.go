package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: "argoproj.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	EventBusGroupVersionKind        = SchemeGroupVersion.WithKind("EventBus")
	EventBusGroupVersionResource    = SchemeGroupVersion.WithResource("eventbus")
	EventSourceGroupVersionKind     = SchemeGroupVersion.WithKind("EventSource")
	EventSourceGroupVersionResource = SchemeGroupVersion.WithResource("eventsources")
	SensorGroupVersionKind          = SchemeGroupVersion.WithKind("Sensor")
	SensorGroupVersionResource      = SchemeGroupVersion.WithResource("sensors")
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func GroupKind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&EventBus{},
		&EventBusList{},
		&EventSource{},
		&EventSourceList{},
		&Sensor{},
		&SensorList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
