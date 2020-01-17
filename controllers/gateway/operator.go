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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwclient "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// the gatewayContext of an operation in the controller.
// the controller creates this gatewayContext each time it picks a Gateway off its queue.
type gatewayContext struct {
	// gateway is the controller object
	gateway *v1alpha1.Gateway
	// updated indicates whether the controller object was updated and needs to be persisted back to k8
	updated bool
	// logger is the logger for a gateway
	logger *logrus.Logger
	// reference to the controller
	controller *Controller
}

// newGatewayContext creates and initializes a new gatewayContext object
func newGatewayContext(gatewayObj *v1alpha1.Gateway, controller *Controller) *gatewayContext {
	gatewayObj = gatewayObj.DeepCopy()
	return &gatewayContext{
		gateway: gatewayObj,
		updated: false,
		logger: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelResourceName: gatewayObj.Name,
				common.LabelNamespace:    gatewayObj.Namespace,
			}).Logger,
		controller: controller,
	}
}

// operate checks the status of gateway resource and takes action based on it.
func (ctx *gatewayContext) operate() error {
	defer ctx.updateGatewayState()

	ctx.logger.WithField(common.LabelPhase, string(ctx.gateway.Status.Phase)).Infoln("operating on the gateway...")

	if err := Validate(ctx.gateway); err != nil {
		ctx.logger.WithError(err).Infoln("invalid gateway object")
		return err
	}

	// check the state of a gateway and take actions accordingly
	switch ctx.gateway.Status.Phase {
	case v1alpha1.NodePhaseNew:
		if err := ctx.createGatewayResources(); err != nil {
			ctx.logger.WithError(err).Errorln("failed to create resources for the gateway")
			ctx.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
			return err
		}
		ctx.logger.Infoln("marking gateway as active")
		ctx.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")

		// Gateway is already running
	case v1alpha1.NodePhaseRunning:
		ctx.logger.Infoln("gateway is running")
		err := ctx.updateGatewayResources()
		if err != nil {
			ctx.logger.WithError(err).Errorln("failed to update resources for the gateway")
			ctx.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
			return err
		}

	// Gateway is in error
	case v1alpha1.NodePhaseError:
		ctx.logger.Errorln("gateway is in error state. checking updates for gateway object...")
		err := ctx.updateGatewayResources()
		if err != nil {
			ctx.logger.WithError(err).Errorln("failed to update resources for the gateway")
			return err
		}
		ctx.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is now active")

	default:
		ctx.logger.WithField(common.LabelPhase, string(ctx.gateway.Status.Phase)).Errorln("unknown gateway phase")
	}
	return nil
}

// updateGatewayState updates the gateway state
func (ctx *gatewayContext) updateGatewayState() {
	defer func() {
		ctx.updated = false
	}()
	if ctx.updated {
		updatedGateway, err := PersistUpdates(ctx.controller.gatewayClient, ctx.gateway, ctx.logger)
		if err != nil {
			ctx.logger.WithError(err).Errorln("failed to persist gateway update")
			return
		}
		ctx.gateway = updatedGateway
		ctx.logger.Infoln("successfully persisted gateway resource update and created K8s event")
	}
}

// mark the gateway phase
func (ctx *gatewayContext) markGatewayPhase(phase v1alpha1.NodePhase, message string) {
	justCompleted := ctx.gateway.Status.Phase != phase
	if justCompleted {
		ctx.logger.WithFields(
			map[string]interface{}{
				"old": string(ctx.gateway.Status.Phase),
				"new": string(phase),
			},
		).Infoln("phase changed")

		ctx.gateway.Status.Phase = phase
		if ctx.gateway.ObjectMeta.Labels == nil {
			ctx.gateway.ObjectMeta.Labels = make(map[string]string)
		}
		if ctx.gateway.Annotations == nil {
			ctx.gateway.Annotations = make(map[string]string)
		}

		ctx.gateway.ObjectMeta.Labels[LabelPhase] = string(phase)
		// add annotations so a resource sensor can watch this gateway.
		ctx.gateway.Annotations[LabelPhase] = string(phase)
	}

	if ctx.gateway.Status.StartedAt.IsZero() {
		ctx.gateway.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}

	ctx.logger.WithFields(
		map[string]interface{}{
			"old": ctx.gateway.Status.Message,
			"new": message,
		},
	).Infoln("phase change message")

	ctx.gateway.Status.Message = message
	ctx.updated = true
}

// PersistUpdates of the gateway resource
func PersistUpdates(client gwclient.Interface, gw *v1alpha1.Gateway, log *logrus.Logger) (*v1alpha1.Gateway, error) {
	gatewayClient := client.ArgoprojV1alpha1().Gateways(gw.ObjectMeta.Namespace)

	updatedGateway, err := gatewayClient.Update(gw)
	if err != nil {
		log.WithError(err).Warn("error updating gateway")
		if errors.IsConflict(err) {
			return nil, err
		}
		log.Info("re-applying updates on latest version and retrying update")
		err = ReapplyUpdates(client, gw)
		if err != nil {
			log.WithError(err).Error("failed to re-apply update")
			return nil, err
		}
	}
	log.WithField(common.LabelPhase, string(updatedGateway.Status.Phase)).Info("gateway state updated successfully")
	return updatedGateway, nil
}

// ReapplyUpdates to gateway resource
func ReapplyUpdates(client gwclient.Interface, gw *v1alpha1.Gateway) error {
	return wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		gatewayClient := client.ArgoprojV1alpha1().Gateways(gw.Namespace)
		g, err := gatewayClient.Update(gw)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		gw = g
		return true, nil
	})
}
