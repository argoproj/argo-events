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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	log *logrus.Logger
	// reference to the gateway-controller-controller
	controller *GatewayController
	// gwrctx is the context to handle child resource
	gwrctx gwResourceCtx
}

// newGatewayOperationCtx creates and initializes a new gOperationCtx object
func newGatewayOperationCtx(gw *v1alpha1.Gateway, controller *GatewayController) *gwOperationCtx {
	gw = gw.DeepCopy()
	return &gwOperationCtx{
		gw:      gw,
		updated: false,
		log: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelGatewayName: gw.Name,
				common.LabelNamespace:   gw.Namespace,
			}).Logger,
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
			goc.gw, err = PersistUpdates(goc.controller.gatewayClientset, goc.gw, goc.log)
			if err != nil {
				goc.log.WithError(err).Error("failed to persist gateway update, escalating...")
				// escalate
				eventType = common.EscalationEventType
			}

			labels[common.LabelEventType] = string(eventType)
			if err := common.GenerateK8sEvent(goc.controller.kubeClientset,
				"persist update",
				eventType,
				"gateway state update",
				goc.gw.Name,
				goc.gw.Namespace,
				goc.controller.Config.InstanceID,
				gateway.Kind,
				labels,
			); err != nil {
				goc.log.WithError(err).Error("failed to create K8s event to log gateway state persist operation")
				return
			}
			goc.log.Info("successfully persisted gateway resource update and created K8s event")
		}
		goc.updated = false
	}()

	goc.log.WithField(common.LabelPhase, string(goc.gw.Status.Phase)).Info("operating on the gateway")

	// check the state of a gateway and take actions accordingly
	switch goc.gw.Status.Phase {
	case v1alpha1.NodePhaseNew:
		err := goc.createGatewayResources()
		if err != nil {
			return err
		}

	// Gateway is in error
	case v1alpha1.NodePhaseError:
		goc.log.Error("gateway is in error state. please check escalated K8 event for the error")
		err := goc.updateGatewayResources()
		if err != nil {
			return err
		}

	// Gateway is already running, do nothing
	case v1alpha1.NodePhaseRunning:
		goc.log.Info("gateway is running")
		err := goc.updateGatewayResources()
		if err != nil {
			return err
		}

	default:
		goc.log.WithField(common.LabelPhase, string(goc.gw.Status.Phase)).Panic("unknown gateway phase.")
	}
	return nil
}

func (goc *gwOperationCtx) createGatewayResources() error {
	err := Validate(goc.gw)
	if err != nil {
		goc.log.WithError(err).Error("gateway validation failed")
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
	goc.log.WithField(common.LabelPodName, pod.Name).Info("gateway pod is created")

	// expose gateway if service is configured
	if goc.gw.Spec.Service != nil {
		svc, err := goc.createGatewayService()
		if err != nil {
			err = errors.Wrap(err, "failed to create gateway service")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
			return err
		}
		goc.log.WithField(common.LabelServiceName, svc.Name).Info("gateway service is created")
	}

	goc.log.Info("marking gateway as active")
	goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")
	return nil
}

func (goc *gwOperationCtx) createGatewayPod() (*corev1.Pod, error) {
	pod, err := goc.gwrctx.newGatewayPod()
	if err != nil {
		goc.log.WithError(err).Error("failed to initialize pod for gateway")
		return nil, err
	}
	pod, err = goc.gwrctx.createGatewayPod(pod)
	if err != nil {
		goc.log.WithError(err).Error("failed to create pod for gateway")
		return nil, err
	}
	return pod, nil
}

func (goc *gwOperationCtx) createGatewayService() (*corev1.Service, error) {
	svc, err := goc.gwrctx.newGatewayService()
	if err != nil {
		goc.log.WithError(err).Error("failed to initialize service for gateway")
		return nil, err
	}
	svc, err = goc.gwrctx.createGatewayService(svc)
	if err != nil {
		goc.log.WithError(err).Error("failed to create service for gateway")
		return nil, err
	}
	return svc, nil
}

func (goc *gwOperationCtx) updateGatewayResources() error {
	err := Validate(goc.gw)
	if err != nil {
		goc.log.WithError(err).Error("gateway validation failed")
		err = errors.Wrap(err, "failed to validate gateway")
		if goc.gw.Status.Phase != v1alpha1.NodePhaseError {
			goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		}
		return err
	}

	_, podChanged, err := goc.updateGatewayPod()
	if err != nil {
		err = errors.Wrap(err, "failed to update gateway pod")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
		return err
	}

	_, svcChanged, err := goc.updateGatewayService()
	if err != nil {
		err = errors.Wrap(err, "failed to update gateway service")
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
		goc.log.WithError(err).Error("failed to get pod for gateway")
		return nil, false, err
	}

	// create a new pod spec
	newPod, err := goc.gwrctx.newGatewayPod()
	if err != nil {
		goc.log.WithError(err).Error("failed to initialize pod for gateway")
		return nil, false, err
	}

	// check if pod spec remained unchanged
	if existingPod != nil {
		if existingPod.Annotations != nil && existingPod.Annotations[common.AnnotationGatewayResourceSpecHashName] == newPod.Annotations[common.AnnotationGatewayResourceSpecHashName] {
			goc.log.WithField(common.LabelPodName, existingPod.Name).Debug("gateway pod spec unchanged")
			return nil, false, nil
		}

		// By now we are sure that the spec changed, so lets go ahead and delete the existing gateway pod.
		goc.log.WithField(common.LabelPodName, existingPod.Name).Info("gateway pod spec changed")

		err := goc.gwrctx.deleteGatewayPod(existingPod)
		if err != nil {
			goc.log.WithError(err).Error("failed to delete pod for gateway")
			return nil, false, err
		}

		goc.log.WithField(common.LabelPodName, existingPod.Name).Info("gateway pod is deleted")
	}

	// Create new pod for updated gateway spec.
	createdPod, err := goc.gwrctx.createGatewayPod(newPod)
	if err != nil {
		goc.log.WithError(err).Error("failed to create pod for gateway")
		return nil, false, err
	}
	goc.log.WithError(err).WithField(common.LabelPodName, newPod.Name).Info("gateway pod is created")

	return createdPod, true, nil
}

func (goc *gwOperationCtx) updateGatewayService() (*corev1.Service, bool, error) {
	// Check if gateway spec has changed for service.
	existingSvc, err := goc.gwrctx.getGatewayService()
	if err != nil {
		goc.log.WithError(err).Error("failed to get service for gateway")
		return nil, false, err
	}

	// create a new service spec
	newSvc, err := goc.gwrctx.newGatewayService()
	if err != nil {
		goc.log.WithError(err).Error("failed to initialize service for gateway")
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
		if existingSvc.Annotations[common.AnnotationGatewayResourceSpecHashName] == newSvc.Annotations[common.AnnotationGatewayResourceSpecHashName] {
			goc.log.WithField(common.LabelServiceName, existingSvc.Name).Debug("gateway service spec unchanged")
			return nil, false, nil
		}

		// service spec changed, delete existing service and create new one
		goc.log.WithField(common.LabelServiceName, existingSvc.Name).Info("gateway service spec changed")

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
		goc.log.WithError(err).Error("failed to create service for gateway")
		return nil, false, err
	}
	goc.log.WithField(common.LabelServiceName, createdSvc.Name).Info("gateway service is created")

	return createdSvc, true, nil
}

// mark the overall gateway phase
func (goc *gwOperationCtx) markGatewayPhase(phase v1alpha1.NodePhase, message string) {
	justCompleted := goc.gw.Status.Phase != phase
	if justCompleted {
		goc.log.WithFields(
			map[string]interface{}{
				"old": string(goc.gw.Status.Phase),
				"new": string(phase),
			},
		).Info("phase changed")

		goc.gw.Status.Phase = phase
		if goc.gw.ObjectMeta.Labels == nil {
			goc.gw.ObjectMeta.Labels = make(map[string]string)
		}
		if goc.gw.Annotations == nil {
			goc.gw.Annotations = make(map[string]string)
		}
		goc.gw.ObjectMeta.Labels[common.LabelGatewayKeyPhase] = string(phase)
		// add annotations so a resource sensor can watch this gateway.
		goc.gw.Annotations[common.LabelGatewayKeyPhase] = string(phase)
	}
	if goc.gw.Status.StartedAt.IsZero() {
		goc.gw.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}
	goc.log.WithFields(
		map[string]interface{}{
			"old": goc.gw.Status.Message,
			"new": message,
		},
	).Info("message")
	goc.gw.Status.Message = message
	goc.updated = true
}
