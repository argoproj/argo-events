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
func (gc *GatewayConfig) GetK8Event(reason string, action v1alpha1.NodePhase, config *ConfigData) *corev1.Event {
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
				common.LabelEventSeen:                "",
				common.LabelResourceName:             gc.gw.Name,
				common.LabelEventType:                string(common.ResourceStateChangeEventType),
				common.LabelGatewayConfigurationName: config.Src,
				common.LabelGatewayName:              gc.Name,
				common.LabelGatewayConfigID:          config.ID,
				common.LabelGatewayConfigTimeID:      config.TimeID,
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

// GatewayCleanup marks configuration as non-active and marks final gateway state
func (gc *GatewayConfig) GatewayCleanup(config *ConfigContext, err error) {
	var event *corev1.Event
	// mark configuration as deactivated so gateway processor client won't run configStopper in case if there
	// was configuration error.
	config.Active = false
	// check if gateway configuration is in error condition.
	if err != nil {
		gc.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("error occurred while running configuration")
		// create k8 event for error state
		event = gc.GetK8Event(err.Error(), v1alpha1.NodePhaseError, config.Data)
	} else {
		// gateway successfully completed/deactivated this configuration.
		gc.Log.Info().Str("config-key", config.Data.Src).Msg("configuration completed")
		// create k8 event for completion state
		event = gc.GetK8Event("configuration completed", v1alpha1.NodePhaseCompleted, config.Data)
	}
	_, err = common.CreateK8Event(event, gc.Clientset)
	if err != nil {
		gc.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to create gateway k8 event")
	}
	CloseChannels(config)
}
