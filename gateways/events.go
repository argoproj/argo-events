package gateways

import (
	"github.com/argoproj/argo-events/common"
	"time"
	"github.com/argoproj/argo-events/pkg/apis/gateway"
	corev1 "k8s.io/api/core/v1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
