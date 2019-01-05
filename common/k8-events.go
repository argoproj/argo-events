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

package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sEventType is the type of event generated to indicate change in state of resource
type K8sEventType string

// Possible values for K8sEventType
var (
	EscalationEventType  K8sEventType = "Escalation"
	StateChangeEventType K8sEventType = "StateChange"
)

const (
	// LabelEventSeen is the label for already seen k8 event
	LabelEventSeen = "event-seen"

	// LabelResourceName is the label for a resource name. It is used while creating K8 event to indicate which component/resource caused the event.
	LabelResourceName = "component"

	// LabelEventType is label for k8 event type
	LabelEventType = "event-type"
)

// CreateK8Event creates a kubernetes event resource
func CreateK8Event(event *corev1.Event, clientset kubernetes.Interface) (*corev1.Event, error) {
	k8Event, err := clientset.CoreV1().Events(event.ObjectMeta.Namespace).Create(event)
	return k8Event, err
}
