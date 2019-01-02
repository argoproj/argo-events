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
	"os"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	client "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/typed/gateway/v1alpha1"
	zlog "github.com/rs/zerolog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	gatewayProcessor = "gateway-processor"
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
		log:        zlog.New(os.Stdout).With().Str("name", gw.Name).Str("namespace", gw.Namespace).Caller().Logger(),
		controller: controller,
	}
}

// operate checks the status of gateway resource and takes action based on it.
func (goc *gwOperationCtx) operate() error {
	// persist updates to gateway resource once we are done operating on it.
	defer goc.persistUpdates()
	goc.log.Info().Msg("operating on the gateway")

	// performs a basic validation on gateway resource.
	err := Validate(goc.gw)
	if err != nil {
		goc.log.Error().Err(err).Msg("gateway validation failed")
		goc.markGatewayPhase(v1alpha1.NodePhaseError, "validation failed")
		return err
	}

	// check the state of a gateway and take actions accordingly
	switch goc.gw.Status.Phase {
	case v1alpha1.NodePhaseNew:
		// Gateway pod has two components,
		// 1) Gateway Server   - Listen events from event source and dispatches the event to gateway client
		// 2) Gateway Client   - Listens for events from gateway server, convert them into cloudevents specification
		//                          compliant events and dispatch them to watchers.
		gatewayPod := &appsv1.Deployment{
			ObjectMeta: goc.gw.Spec.DeploySpec.ObjectMeta,
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: goc.gw.Spec.DeploySpec.Labels,
				},
				Template: corev1.PodTemplateSpec{
					Spec:       goc.gw.Spec.DeploySpec.Spec,
					ObjectMeta: goc.gw.Spec.DeploySpec.ObjectMeta,
				},
			},
		}
		gatewayPod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
		}
		gatewayPod.Spec.Template.Spec.Containers = *goc.getContainersForGatewayPod()
		gatewayPod.Name = common.DefaultGatewayPodName(goc.gw.Name)

		// we can now create the gateway deployment.
		// depending on user configuration gateway will be exposed outside the cluster or intra-cluster.
		_, err = goc.controller.kubeClientset.AppsV1().Deployments(goc.gw.Namespace).Create(gatewayPod)
		if err != nil {
			goc.log.Error().Err(err).Msg("failed gateway deployment")
			goc.markGatewayPhase(v1alpha1.NodePhaseError, fmt.Sprintf("failed gateway deployment. err: %s", err))
			return err
		}

		goc.log.Info().Str("deployment-name", common.DefaultGatewayPodName(goc.gw.Name)).Msg("gateway deployment created")

		// expose gateway if service is configured
		if goc.gw.Spec.ServiceSpec != nil {
			svc, err := goc.createGatewayService()
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

	default:
		goc.log.Panic().Str("phase", string(goc.gw.Status.Phase)).Msg("unknown gateway phase.")
	}
	return nil
}

// Creates a service that exposes gateway.
func (goc *gwOperationCtx) createGatewayService() (*corev1.Service, error) {
	gatewayService := goc.gw.Spec.ServiceSpec
	gatewayService.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(goc.gw, v1alpha1.SchemaGroupVersionKind),
	}
	svc, err := goc.controller.kubeClientset.CoreV1().Services(goc.gw.Namespace).Create(gatewayService)
	return svc, err
}

// persist the updates to the gateway resource
func (goc *gwOperationCtx) persistUpdates() {
	var err error
	if !goc.updated {
		return
	}
	gatewayClient := goc.controller.gatewayClientset.ArgoprojV1alpha1().Gateways(goc.gw.ObjectMeta.Namespace)
	goc.gw, err = gatewayClient.Update(goc.gw)
	if err != nil {
		goc.log.Warn().Err(err).Msg("error updating gateway")
		if errors.IsConflict(err) {
			return
		}
		goc.log.Info().Msg("re-applying updates on latest version and retrying update")
		err = goc.reapplyUpdate(gatewayClient)
		if err != nil {
			goc.log.Error().Err(err).Msg("failed to re-apply update")
			return
		}
	}
	goc.log.Info().Str("gateway-phase", string(goc.gw.Status.Phase)).Msg("gateway state updated successfully")
	time.Sleep(1 * time.Second)
}

// reapply the updates to gateway resource
func (goc *gwOperationCtx) reapplyUpdate(gatewayClient client.GatewayInterface) error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		g, err := gatewayClient.Get(goc.gw.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		g.Status = goc.gw.Status
		goc.gw, err = gatewayClient.Update(g)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	})
}

// mark the overall gateway phase
func (goc *gwOperationCtx) markGatewayPhase(phase v1alpha1.NodePhase, message string) {
	justCompleted := goc.gw.Status.Phase != phase
	if justCompleted {
		goc.log.Info().Str("old-phase", string(goc.gw.Status.Phase)).Str("new-phase", string(phase))
		goc.gw.Status.Phase = phase
		goc.updated = true
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
		goc.updated = true
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
	}
	var containers []corev1.Container
	for i, container := range goc.gw.Spec.DeploySpec.Spec.Containers {
		container.Env = append(container.Env, envVars...)
		containers[i] = container
	}
	return &containers
}
