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

package gateway

import (
	"fmt"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/gateway"

	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	zlog "github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// the context of an operation on a gateway-controller.
// the gateway-controller-controller creates this context each time it picks a Gateway off its queue.
type gwOperationCtx struct {
	// gw is the gateway-controller object
	gw *v1alpha1.Gateway
	// updated indicates whether the gateway-controller object was updated and needs to be persisted back to k8
	updated bool
	// log is the logger for a gateway
	log zlog.Logger
	// reference to the gateway-controller-controller
	controller *GatewayController
	// crctx is the context to handle child resource
	crctx controllerscommon.ChildResourceContext
}

// newGatewayOperationCtx creates and initializes a new gOperationCtx object
func newGatewayOperationCtx(gw *v1alpha1.Gateway, controller *GatewayController) *gwOperationCtx {
	return &gwOperationCtx{
		gw:         gw.DeepCopy(),
		updated:    false,
		log:        common.GetLoggerContext(common.LoggerConf()).Str("name", gw.Name).Str("namespace", gw.Namespace).Logger(),
		controller: controller,
		crctx: controllerscommon.ChildResourceContext{
			SchemaGroupVersionKind:            v1alpha1.SchemaGroupVersionKind,
			LabelOwnerName:                    common.LabelGatewayName,
			LabelKeyOwnerControllerInstanceID: common.LabelKeyGatewayControllerInstanceID,
			AnnotationOwnerResourceHashName:   common.AnnotationSensorResourceSpecHashName,
			InstanceID:                        controller.Config.InstanceID,
		},
	}
}

// operate checks the status of gateway resource and takes action based on it.
func (goc *gwOperationCtx) operate() error {
	defer func() {
		if goc.updated {
			var err error
			eventType := common.StateChangeEventType
			labels := map[string]string{
				common.LabelGatewayName:                    goc.gw.Name,
				common.LabelGatewayKeyPhase:                string(goc.gw.Status.Phase),
				common.LabelKeyGatewayControllerInstanceID: goc.controller.Config.InstanceID,
				common.LabelOperation:                      "persist_gateway_state",
			}
			goc.gw, err = PersistUpdates(goc.controller.gatewayClientset, goc.gw, &goc.log)
			if err != nil {
				goc.log.Error().Err(err).Msg("failed to persist gateway update, escalating...")
				// escalate
				eventType = common.EscalationEventType
			}

			labels[common.LabelEventType] = string(eventType)
			if err := common.GenerateK8sEvent(goc.controller.kubeClientset, "persist update", eventType, "gateway state update", goc.gw.Name, goc.gw.Namespace,
				goc.controller.Config.InstanceID, gateway.Kind, labels); err != nil {
				goc.log.Error().Err(err).Msg("failed to create K8s event to log gateway state persist operation")
				return
			}
			goc.log.Info().Msg("successfully persisted gateway resource update and created K8s event")
		}
		goc.updated = false
	}()

	goc.log.Info().Str("phase", string(goc.gw.Status.Phase)).Msg("operating on the gateway")

	// check the state of a gateway and take actions accordingly
	switch goc.gw.Status.Phase {
	case v1alpha1.NodePhaseNew:
		err := goc.createGatewayResources()
		if err != nil {
			return err
		}

	// Gateway is in error
	case v1alpha1.NodePhaseError:
		goc.log.Error().Str("gateway-name", goc.gw.Name).Msg("gateway is in error state. please check escalated K8 event for the error")

		err := goc.updateGatewayResources()
		if err != nil {
			return err
		}

	// Gateway is already running, do nothing
	case v1alpha1.NodePhaseRunning:
		goc.log.Info().Str("gateway-name", goc.gw.Name).Msg("gateway is running")

		err := goc.updateGatewayResources()
		if err != nil {
			return err
		}

	default:
		goc.log.Panic().Str("gateway-name", goc.gw.Name).Str("phase", string(goc.gw.Status.Phase)).Msg("unknown gateway phase.")
	}
	return nil
}

func (goc *gwOperationCtx) createGatewayResources() error {
	err := Validate(goc.gw)
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("gateway validation failed")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, "validation failed")
		return err
	}
	// Gateway pod has two components,
	// 1) Gateway Server   - Listen events from event source and dispatches the event to gateway client
	// 2) Gateway Client   - Listens for events from gateway server, convert them into cloudevents specification
	//                          compliant events and dispatch them to watchers.
	pod, err := goc.createGatewayPod()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to create pod for gateway")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway pod. err: %s", err))
		return err
	}
	goc.log.Info().Str("pod-name", pod.Name).Msg("gateway pod is created")

	// expose gateway if service is configured
	if goc.gw.Spec.ServiceSpec != nil {
		svc, err := goc.createGatewayService()
		if err != nil {
			goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to create service for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway service. err: %s", err))
			return err
		}
		goc.log.Info().Str("svc-name", svc.Name).Msg("gateway service is created")
	}

	goc.log.Info().Str("gateway-name", goc.gw.Name).Msg("marking gateway as active")
	goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")
	return nil
}

func (goc *gwOperationCtx) updateGatewayResources() error {
	err := Validate(goc.gw)
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("gateway validation failed")
		if goc.gw.Status.Phase != v1alpha1.NodePhaseError {
			goc.markGatewayPhase(v1alpha1.NodePhaseError, "validation failed")
		}
		return err
	}

	deleted := false
	created := false

	pod, err := goc.getGatewayPod()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to get pod for gateway")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to get gateway pod. err: %s", err))
		return err
	}
	if pod != nil {
		newPod, err := goc.newGatewayPod()
		if err != nil {
			goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to initialize pod for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to initialize gateway pod. err: %s", err))
			return err
		}
		if pod.Annotations == nil || pod.Annotations[common.AnnotationSensorResourceSpecHashName] != newPod.Annotations[common.AnnotationSensorResourceSpecHashName] {
			goc.log.Info().Str("pod-name", pod.Name).Msg("gateway pod spec changed")
			err := goc.controller.kubeClientset.CoreV1().Pods(goc.gw.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err != nil {
				goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to delete pod for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to delete gateway pod. err: %s", err))
				return err
			}
			goc.log.Info().Str("pod-name", pod.Name).Msg("gateway pod is deleted")
			deleted = true
		}
	} else {
		goc.log.Info().Str("gateway-name", goc.gw.Name).Msg("gateway pod has been deleted")
		pod, err := goc.createGatewayPod()
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to create pod for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway pod. err: %s", err))
			return err
		}
		goc.log.Info().Str("pod-name", pod.Name).Msg("gateway pod is created")
		created = true
	}

	svc, err := goc.getGatewayService()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to get service for gateway")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to get gateway service. err: %s", err))
		return err
	}
	if svc != nil {
		var newSvc *corev1.Service
		if goc.gw.Spec.ServiceSpec != nil {
			newSvc, err = goc.newGatewayService()
			if err != nil {
				goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to initialize service for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to initialize gateway service. err: %s", err))
				return err
			}
		}
		if newSvc == nil || svc.Annotations == nil || svc.Annotations[common.AnnotationSensorResourceSpecHashName] != newSvc.Annotations[common.AnnotationSensorResourceSpecHashName] {
			goc.log.Info().Str("svc-name", svc.Name).Msg("gateway service spec changed")
			err := goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
			if err != nil {
				goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to delete service for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to delete gateway service. err: %s", err))
				return err
			}
			goc.log.Info().Str("svc-name", svc.Name).Msg("gateway service is deleted")
			deleted = true
		}
	} else {
		if goc.gw.Spec.ServiceSpec != nil {
			goc.log.Info().Str("gateway-name", goc.gw.Name).Msg("gateway service has been deleted")
			svc, err := goc.createGatewayService()
			if err != nil {
				goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to create service for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway service. err: %s", err))
				return err
			}
			goc.log.Info().Str("svc-name", svc.Name).Msg("gateway service is created")
			created = true
		}
	}

	if created && !deleted && goc.gw.Status.Phase != v1alpha1.NodePhaseRunning {
		goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")
	}
	return nil
}

// mark the overall gateway phase
func (goc *gwOperationCtx) markGatewayPhase(phase v1alpha1.NodePhase, message string) {
	justCompleted := goc.gw.Status.Phase != phase
	if justCompleted {
		goc.log.Info().Str("old-phase", string(goc.gw.Status.Phase)).Str("new-phase", string(phase))
		goc.gw.Status.Phase = phase
		if goc.gw.ObjectMeta.Labels == nil {
			goc.gw.ObjectMeta.Labels = make(map[string]string)
		}
		if goc.gw.Annotations == nil {
			goc.gw.Annotations = make(map[string]string)
		}
		goc.gw.ObjectMeta.Labels[common.LabelSensorKeyPhase] = string(phase)
		// add annotations so a resource sensor can watch this gateway.
		goc.gw.Annotations[common.LabelGatewayKeyPhase] = string(phase)
	}
	if goc.gw.Status.StartedAt.IsZero() {
		goc.gw.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}
	goc.log.Info().Str("old-message", string(goc.gw.Status.Message)).Str("new-message", message)
	goc.gw.Status.Message = message
	goc.updated = true
}
