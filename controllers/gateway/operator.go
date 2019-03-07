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
	"encoding/json"
	"fmt"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/gateway"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	zlog "github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
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
}

// newGatewayOperationCtx creates and initializes a new gOperationCtx object
func newGatewayOperationCtx(gw *v1alpha1.Gateway, controller *GatewayController) *gwOperationCtx {
	return &gwOperationCtx{
		gw:         gw.DeepCopy(),
		updated:    false,
		log:        common.GetLoggerContext(common.LoggerConf()).Str("name", gw.Name).Str("namespace", gw.Namespace).Logger(),
		controller: controller,
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
		// perform one-time gateway validation
		// non nil err indicates failed validation
		// we do not want to requeue a gateway in this case
		// since validation will fail every time
		err := Validate(goc.gw)
		if err != nil {
			goc.log.Error().Err(err).Msg("gateway validation failed")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, "validation failed")
			return err
		}

		// Gateway pod has two components,
		// 1) Gateway Server   - Listen events from event source and dispatches the event to gateway client
		// 2) Gateway Client   - Listens for events from gateway server, convert them into cloudevents specification
		//                          compliant events and dispatch them to watchers.

		pod, err := goc.newGatewayPod()
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to initialize pod for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to initialize gateway pod. err: %s", err))
			return err
		}
		pod, err = goc.controller.kubeClientset.CoreV1().Pods(goc.gw.Namespace).Create(pod)
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to create pod for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway pod. err: %s", err))
			return err
		}
		goc.log.Info().Str("pod-name", pod.ObjectMeta.Name).Msg("gateway pod created")

		// expose gateway if service is configured
		if goc.gw.Spec.ServiceSpec != nil {
			svc, err := goc.newGatewayService()
			if err != nil {
				goc.log.Error().Err(err).Msg("failed to initialize service for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to initialize gateway service. err: %s", err))
				return err
			}
			svc, err = goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Create(svc)
			if err != nil {
				goc.log.Error().Err(err).Msg("failed to create service for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway service. err: %s", err))
				return err
			}
			goc.log.Info().Str("svc-name", svc.ObjectMeta.Name).Msg("gateway service is created")
		}
		goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")

	// Gateway is in error
	case v1alpha1.NodePhaseError:
		goc.log.Error().Msg("gateway is in error state. please check escalated K8 event for the error")

	// Gateway is already running, do nothing
	case v1alpha1.NodePhaseRunning:
		goc.log.Info().Msg("gateway is running")

		newPod, err := goc.newGatewayPod()
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to initialize pod for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to initialize gateway pod. err: %s", err))
			return err
		}
		pod, err := goc.getGatewayPod()
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to get pod for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to get gateway pod. err: %s", err))
			return err
		}
		if pod != nil {
			if pod.Annotations != nil && pod.Annotations[common.AnnotationGatewayResourceHashName] != newPod.Annotations[common.AnnotationGatewayResourceHashName] {
				goc.log.Info().Str("pod-name", pod.ObjectMeta.Name).Msg("gateway pod spec changed")
				err := goc.controller.kubeClientset.CoreV1().Pods(goc.gw.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
				if err != nil {
					goc.log.Error().Err(err).Msg("failed to delete pod for gateway")
					goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to delete gateway pod. err: %s", err))
					return err
				}
			}
		} else {
			goc.log.Info().Str("gateway-name", goc.gw.ObjectMeta.Name).Msg("gateway pod is deleted")
			pod, err = goc.controller.kubeClientset.CoreV1().Pods(goc.gw.Namespace).Create(newPod)
			if err != nil {
				goc.log.Error().Err(err).Msg("failed to create pod for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway pod. err: %s", err))
				return err
			}
			goc.log.Info().Str("pod-name", pod.ObjectMeta.Name).Msg("gateway pod created")
		}

		var newSvc *corev1.Service
		if goc.gw.Spec.ServiceSpec != nil {
			newSvc, err = goc.newGatewayService()
			if err != nil {
				goc.log.Error().Err(err).Msg("failed to initialize service for gateway")
				goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to initialize gateway service. err: %s", err))
				return err
			}
		}
		svc, err := goc.getGatewayService()
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to get service for gateway")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to get gateway service. err: %s", err))
			return err
		}
		if svc != nil {
			if newSvc == nil || svc.Annotations != nil && svc.Annotations[common.AnnotationGatewayResourceHashName] != newSvc.Annotations[common.AnnotationGatewayResourceHashName] {
				goc.log.Info().Str("svc-name", svc.ObjectMeta.Name).Msg("gateway service spec changed")
				err := goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Delete(svc.Name, &metav1.DeleteOptions{})
				if err != nil {
					goc.log.Error().Err(err).Msg("failed to delete service for gateway")
					goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to delete gateway service. err: %s", err))
					return err
				}
			}
		} else {
			if newSvc != nil {
				goc.log.Info().Str("gateway-name", goc.gw.ObjectMeta.Name).Msg("gateway service is deleted")
				svc, err = goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Create(newSvc)
				if err != nil {
					goc.log.Error().Err(err).Msg("failed to create service for gateway")
					goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed to create gateway service. err: %s", err))
					return err
				}
				goc.log.Info().Str("svc-name", svc.ObjectMeta.Name).Msg("gateway service is created")
			}
		}

	default:
		goc.log.Panic().Str("phase", string(goc.gw.Status.Phase)).Msg("unknown gateway phase.")
	}
	return nil
}

// gatewayNameReq returns label requirement of gateway name
func (goc *gwOperationCtx) gatewayNameReq() (*labels.Requirement, error) {
	return labels.NewRequirement(common.LabelGatewayName, selection.Equals, []string{goc.gw.Name})
}

// getGatewayService returns the service of gateway
func (goc *gwOperationCtx) getGatewayService() (*corev1.Service, error) {
	req, err := goc.gatewayNameReq()
	if err != nil {
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*req)
	svcs, err := goc.controller.svcInformer.Lister().Services(goc.gw.Namespace).List(labelSelector)
	if err != nil {
		return nil, err
	}
	if len(svcs) == 0 {
		return nil, nil
	}
	return svcs[0], nil
}

// createGatewayPod creates a service that exposes gateway.
func (goc *gwOperationCtx) newGatewayService() (*corev1.Service, error) {
	gatewayService := goc.gw.Spec.ServiceSpec.DeepCopy()
	gatewayService.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
	}
	err := goc.setGatewayLabels(&gatewayService.ObjectMeta)
	if err != nil {
		return nil, err
	}
	hash, err := goc.getGatewayResourceHash(gatewayService)
	if err != nil {
		return nil, err
	}
	if gatewayService.ObjectMeta.Annotations == nil {
		gatewayService.ObjectMeta.Annotations = make(map[string]string)
	}
	gatewayService.ObjectMeta.Annotations[common.AnnotationGatewayResourceHashName] = hash
	return gatewayService, nil
}

// getGatewayPod returns the pod of gateway
func (goc *gwOperationCtx) getGatewayPod() (*corev1.Pod, error) {
	req, err := goc.gatewayNameReq()
	if err != nil {
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*req)
	pods, err := goc.controller.podInformer.Lister().Pods(goc.gw.Namespace).List(labelSelector)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, nil
	}
	return pods[0], nil
}

// createGatewayPod creates a pod of gateway
func (goc *gwOperationCtx) newGatewayPod() (*corev1.Pod, error) {
	gatewayPod := goc.gw.Spec.DeploySpec.DeepCopy()
	gatewayPod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
	}
	if goc.gw.Spec.DeploySpec.Name == "" && goc.gw.Spec.DeploySpec.GenerateName == "" {
		gatewayPod.GenerateName = goc.gw.Name
	}
	gatewayPod.Spec.Containers = *goc.getContainersForGatewayPod()
	err := goc.setGatewayLabels(&gatewayPod.ObjectMeta)
	if err != nil {
		return nil, err
	}
	return gatewayPod, nil
}

// setGatewayLabels sets labels on a given resource for the gateway
func (goc *gwOperationCtx) setGatewayLabels(meta *metav1.ObjectMeta) error {
	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}
	meta.Labels[common.LabelGatewayName] = goc.gw.Name
	meta.Labels[common.LabelKeyGatewayControllerInstanceID] = goc.controller.Config.InstanceID
	return nil
}

// getGatewayResourceHash returns the resource hash on a given resource for the gateway
func (goc *gwOperationCtx) getGatewayResourceHash(obj runtime.Object) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resource")
	}
	return common.Hasher(string(b)), nil
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
		if goc.gw.ObjectMeta.Annotations == nil {
			goc.gw.ObjectMeta.Annotations = make(map[string]string)
		}
		goc.gw.ObjectMeta.Labels[common.LabelSensorKeyPhase] = string(phase)
		// add annotations so a resource sensor can watch this gateway.
		goc.gw.ObjectMeta.Annotations[common.LabelGatewayKeyPhase] = string(phase)
	}
	if goc.gw.Status.StartedAt.IsZero() {
		goc.gw.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}
	goc.log.Info().Str("old-message", string(goc.gw.Status.Message)).Str("new-message", message)
	goc.gw.Status.Message = message
	goc.updated = true
}

// containers required for gateway deployment
func (goc *gwOperationCtx) getContainersForGatewayPod() *[]corev1.Container {
	// env variables
	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarGatewayNamespace,
			Value: goc.gw.Namespace,
		},
		{
			Name:  common.EnvVarGatewayEventSourceConfigMap,
			Value: goc.gw.Spec.ConfigMap,
		},
		{
			Name:  common.EnvVarGatewayName,
			Value: goc.gw.Name,
		},
		{
			Name:  common.EnvVarGatewayControllerInstanceID,
			Value: goc.controller.Config.InstanceID,
		},
		{
			Name:  common.EnvVarGatewayControllerName,
			Value: common.DefaultGatewayControllerDeploymentName,
		},
		{
			Name:  common.EnvVarGatewayServerPort,
			Value: goc.gw.Spec.ProcessorPort,
		},
	}
	containers := make([]corev1.Container, len(goc.gw.Spec.DeploySpec.Spec.Containers))
	for i, container := range goc.gw.Spec.DeploySpec.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		containers[i] = container
	}
	return &containers
}
