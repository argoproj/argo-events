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
	"k8s.io/client-go/kubernetes"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sEventType is the type of event generated to indicate change in state of resource
type K8sEventType string

// Possible values for K8sEventType
var (
	EscalationEventType       K8sEventType = "Escalation"
	StateChangeEventType      K8sEventType = "StateChange"
	OperationFailureEventType K8sEventType = "OperationFailed"
	OperationSuccessEventType K8sEventType = "OperationSuccessful"
)

const (
	// LabelEventType is label for k8 event type
	LabelEventType = "event-type"
)

// GenerateK8sEvent generates a kubernetes event
func GenerateK8sEvent(clientset kubernetes.Interface, reason string, eventType K8sEventType, action string, name, namespace, instanceId, kind string, labels map[string]string) error {
	event := &corev1.Event{
		Reason: reason,
		Type:   string(eventType),
		Action: action,
		EventTime: metav1.MicroTime{
			Time: time.Now(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: name + "-",
			Labels:       labels,
		},
		InvolvedObject: corev1.ObjectReference{
			Namespace: namespace,
			Name:      name,
			Kind:      kind,
		},
		Source: corev1.EventSource{
			Component: name,
		},
		ReportingInstance:   instanceId,
		ReportingController: name,
	}

	if _, err := clientset.CoreV1().Events(namespace).Create(event); err != nil {
		return err
	}
	return nil
}
