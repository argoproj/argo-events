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

package gateways

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// GetK8Event returns a kubernetes event.
func (gc *GatewayConfig) GetK8Event(reason string, action v1alpha1.NodePhase, config *EventSourceData) *corev1.Event {
	return &corev1.Event{
		Reason: reason,
		Type:   string(common.ResourceStateChangeEventType),
		Action: string(action),
		EventTime: metav1.MicroTime{
			Time: time.Now(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    gc.gw.Namespace,
			GenerateName: gc.gw.Name + "-",
			Labels: map[string]string{
				common.LabelEventSeen:              "",
				common.LabelResourceName:           gc.gw.Name,
				common.LabelEventType:              string(common.ResourceStateChangeEventType),
				common.LabelGatewayEventSourceName: config.Src,
				common.LabelGatewayName:            gc.Name,
				common.LabelGatewayEventSourceID:   config.ID,
			},
		},
		InvolvedObject: corev1.ObjectReference{
			Namespace: gc.gw.Namespace,
			Name:      gc.gw.Name,
			Kind:      gateway.Kind,
		},
		Source: corev1.EventSource{
			Component: gc.gw.Name,
		},
		ReportingInstance:   gc.controllerInstanceID,
		ReportingController: gc.gw.Name,
	}
}
