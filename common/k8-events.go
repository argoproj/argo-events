package common

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

// K8Event abstracts kubernetes event.
type K8Event struct {
	// The object that this event is about.
	Name string
	// Namespace where event should be created
	Namespace string
	// What action was taken/failed regarding to the Regarding object.
	Action string
	// This should be a short, machine understandable string that gives the reason
	// for the transition into the object's current status.
	Reason string
	// Kind of component generating this event
	Kind string
	// Name of the controller that emitted this Event
	ReportingController string
	// ID of the controller instance
	ReportingInstance string
	// Type of this event (Normal, Warning), new types could be added in the future
	Type string
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects.
	Labels map[string]string
}

// CreateK8Event returns a kubernetes event object
func GetK8Event(event *K8Event) *corev1.Event {
	return &corev1.Event{
		Reason: event.Reason,
		Type:   event.Type,
		Action: event.Action,
		EventTime: metav1.MicroTime{
			Time: time.Now(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    event.Namespace,
			GenerateName: event.Name + "-",
			Labels:       event.Labels,
		},
		InvolvedObject: corev1.ObjectReference{
			Namespace: event.Namespace,
			Name:      event.Name,
			Kind:      event.Kind,
		},
		Source: corev1.EventSource{
			Component: event.Name,
		},
		ReportingInstance:   event.ReportingInstance,
		ReportingController: event.ReportingController,
	}
}

// CreateK8Event creates a kubernetes event resource
func CreateK8Event(event *corev1.Event, clientset kubernetes.Interface) (*corev1.Event, error) {
	k8Event, err := clientset.CoreV1().Events(event.ObjectMeta.Namespace).Create(event)
	return k8Event, err
}
