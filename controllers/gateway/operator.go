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
	"time"

	"github.com/pkg/errors"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway"
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
	// gwrctx is the context to handle child resource
	gwrctx gwResourceCtx
}

// newGatewayOperationCtx creates and initializes a new gOperationCtx object
func newGatewayOperationCtx(gw *v1alpha1.Gateway, controller *GatewayController) *gwOperationCtx {
	gw = gw.DeepCopy()
	return &gwOperationCtx{
		gw:         gw,
		updated:    false,
		log:        common.GetLoggerContext(common.LoggerConf()).Str("name", gw.Name).Str("namespace", gw.Namespace).Logger(),
		controller: controller,
		gwrctx:     NewGatewayResourceContext(gw, controller),
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
		err = errors.Wrap(err, "failed to validate gateway")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		return err
	}
	// Gateway pod has two components,
	// 1) Gateway Server   - Listen events from event source and dispatches the event to gateway client
	// 2) Gateway Client   - Listens for events from gateway server, convert them into cloudevents specification
	//                          compliant events and dispatch them to watchers.
	pod, err := goc.createGatewayPod()
	if err != nil {
		err = errors.Wrap(err, "failed to create gateway pod")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		return err
	}
	goc.log.Info().Str("pod-name", pod.Name).Msg("gateway pod is created")

	// expose gateway if service is configured
	if goc.gw.Spec.ServiceSpec != nil {
		svc, err := goc.createGatewayService()
		if err != nil {
			err = errors.Wrap(err, "failed to create gateway service")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
			return err
		}
		goc.log.Info().Str("svc-name", svc.Name).Msg("gateway service is created")
	}

	goc.log.Info().Str("gateway-name", goc.gw.Name).Msg("marking gateway as active")
	goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")
	return nil
}

func (goc *gwOperationCtx) createGatewayPod() (*corev1.Pod, error) {
	pod, err := goc.gwrctx.newGatewayPod()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to initialize pod for gateway")
		return nil, err
	}
	pod, err = goc.gwrctx.createGatewayPod(pod)
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to create pod for gateway")
		return nil, err
	}
	return pod, nil
}

func (goc *gwOperationCtx) createGatewayService() (*corev1.Service, error) {
	svc, err := goc.gwrctx.newGatewayService()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to initialize service for gateway")
		return nil, err
	}
	svc, err = goc.gwrctx.createGatewayService(svc)
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to create service for gateway")
		return nil, err
	}
	return svc, nil
}

func (goc *gwOperationCtx) updateGatewayResources() error {
	err := Validate(goc.gw)
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("gateway validation failed")
		err = errors.Wrap(err, "failed to validate gateway")
		if goc.gw.Status.Phase != v1alpha1.NodePhaseError {
			goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		}
		return err
	}

	_, podChanged, err := goc.updateGatewayPod()
	if err != nil {
		err = errors.Wrap(err, "failed to validate gateway pod")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		return err
	}

	_, svcChanged, err := goc.updateGatewayService()
	if err != nil {
		err = errors.Wrap(err, "failed to validate gateway service")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		return err
	}

	if goc.gw.Status.Phase != v1alpha1.NodePhaseRunning && (podChanged || svcChanged) {
		goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")
	}

	return nil
}

func (goc *gwOperationCtx) updateGatewayPod() (*corev1.Pod, bool, error) {
	// Check if gateway spec has changed for pod.
	existingPod, err := goc.gwrctx.getGatewayPod()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to get pod for gateway")
		return nil, false, err
	}

	// create a new pod spec
	newPod, err := goc.gwrctx.newGatewayPod()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to initialize pod for gateway")
		return nil, false, err
	}

	// check if pod spec remained unchanged
	if existingPod != nil {
		if existingPod.Annotations != nil && existingPod.Annotations[common.AnnotationSensorResourceSpecHashName] == newPod.Annotations[common.AnnotationSensorResourceSpecHashName] {
			goc.log.Debug().Str("gateway-name", goc.gw.Name).Str("pod-name", existingPod.Name).Msg("gateway pod spec unchanged")
			return nil, false, nil
		}

		// By now we are sure that the spec changed, so lets go ahead and delete the exisitng gateway pod.
		goc.log.Info().Str("pod-name", existingPod.Name).Msg("gateway pod spec changed")

		err := goc.gwrctx.deleteGatewayPod(existingPod)
		if err != nil {
			goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to delete pod for gateway")
			return nil, false, err
		}

		goc.log.Info().Str("pod-name", existingPod.Name).Msg("gateway pod is deleted")
	}

	// Create new pod for updated gateway spec.
	createdPod, err := goc.gwrctx.createGatewayPod(newPod)
	if err != nil {
		goc.log.Error().Err(err).Msg("failed to create pod for gateway")
		return nil, false, err
	}
	goc.log.Info().Str("gateway-name", goc.gw.Name).Str("pod-name", newPod.Name).Msg("gateway pod is created")

	return createdPod, true, nil
}

func (goc *gwOperationCtx) updateGatewayService() (*corev1.Service, bool, error) {
	// Check if gateway spec has changed for service.
	existingSvc, err := goc.gwrctx.getGatewayService()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to get service for gateway")
		return nil, false, err
	}

	// create a new service spec
	newSvc, err := goc.gwrctx.newGatewayService()
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to initialize service for gateway")
		return nil, false, err
	}

	if existingSvc != nil {
		// updated spec doesn't have service defined, delete existing service.
		if newSvc == nil {
			if err := goc.gwrctx.deleteGatewayService(existingSvc); err != nil {
				return nil, false, err
			}
			return nil, true, nil
		}

		// check if service spec remained unchanged
		if existingSvc.Annotations[common.AnnotationSensorResourceSpecHashName] == newSvc.Annotations[common.AnnotationSensorResourceSpecHashName] {
			goc.log.Debug().Str("gateway-name", goc.gw.Name).Str("service-name", existingSvc.Name).Msg("gateway service spec unchanged")
			return nil, false, nil
		}

		// service spec changed, delete existing service and create new one
		goc.log.Info().Str("gateway-name", goc.gw.Name).Str("service-name", existingSvc.Name).Msg("gateway service spec changed")

		if err := goc.gwrctx.deleteGatewayService(existingSvc); err != nil {
			return nil, false, err
		}
	} else if newSvc == nil {
		// gateway service doesn't exist originally
		return nil, false, nil
	}

	// change createGatewayService to take a service spec
	createdSvc, err := goc.gwrctx.createGatewayService(newSvc)
	if err != nil {
		goc.log.Error().Err(err).Str("gateway-name", goc.gw.Name).Msg("failed to create service for gateway")
		return nil, false, err
	}
	goc.log.Info().Str("svc-name", newSvc.Name).Msg("gateway service is created")

	return createdSvc, true, nil
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
