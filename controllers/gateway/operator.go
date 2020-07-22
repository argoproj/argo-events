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
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwclient "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"go.uber.org/zap"
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
	logger *zap.Logger
	// reference to the controller
	controller *Controller
}

// newGatewayContext creates and initializes a new gatewayContext object
func newGatewayContext(gatewayObj *v1alpha1.Gateway, controller *Controller) *gatewayContext {
	gatewayObj = gatewayObj.DeepCopy()
	return &gatewayContext{
		gateway: gatewayObj,
		updated: false,
		logger: logging.NewArgoEventsLogger().With(
			common.LabelResourceName, gatewayObj.Name).With(logging.LabelNamespace, gatewayObj.Namespace).Desugar(),
		controller: controller,
	}
}

// operate checks the status of gateway resource and takes action based on it.
func (ctx *gatewayContext) operate() error {
	defer ctx.updateGatewayState()

	ctx.logger.Info("operating on the gateway...", zap.Any(logging.LabelPhase, string(ctx.gateway.Status.Phase)))

	if err := Validate(ctx.gateway); err != nil {
		ctx.logger.Info("invalid gateway object", zap.Error(err))
		return err
	}

	// check the state of a gateway and take actions accordingly
	switch ctx.gateway.Status.Phase {
	case v1alpha1.NodePhaseNew:
		if err := ctx.createGatewayResources(); err != nil {
			ctx.logger.Error("failed to create resources for the gateway", zap.Error(err))
			ctx.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
			return err
		}
		ctx.logger.Info("marking gateway as active")
		ctx.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")

	case v1alpha1.NodePhaseRunning:
		ctx.logger.Info("gateway is running")
		err := ctx.updateGatewayResources()
		if err != nil {
			ctx.logger.Error("failed to update resources for the gateway", zap.Error(err))
			ctx.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
			return err
		}

	case v1alpha1.NodePhaseError:
		ctx.logger.Error("gateway is in error state. checking updates for gateway object...")
		err := ctx.updateGatewayResources()
		if err != nil {
			ctx.logger.Error("failed to update resources for the gateway", zap.Error(err))
			return err
		}
		ctx.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is now active")

	default:
		ctx.logger.Error("unknown gateway phase", zap.Any(logging.LabelPhase, string(ctx.gateway.Status.Phase)))
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
			ctx.logger.Error("failed to persist gateway update", zap.Error(err))
			return
		}
		ctx.gateway = updatedGateway
		ctx.logger.Info("successfully persisted gateway resource update and created K8s event")
	}
}

// mark the gateway phase
func (ctx *gatewayContext) markGatewayPhase(phase v1alpha1.NodePhase, message string) {
	justCompleted := ctx.gateway.Status.Phase != phase
	if justCompleted {
		ctx.logger.Info("phase changed", zap.Any("old", string(ctx.gateway.Status.Phase)), zap.Any("new", string(phase)))

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

	ctx.logger.Info("phase change message", zap.Any("old", ctx.gateway.Status.Message), zap.Any("new", message))

	ctx.gateway.Status.Message = message
	ctx.updated = true
}

// PersistUpdates persists the updates for the gateway object.
func PersistUpdates(client gwclient.Interface, gw *v1alpha1.Gateway, log *zap.Logger) (*v1alpha1.Gateway, error) {
	gatewayClient := client.ArgoprojV1alpha1().Gateways(gw.ObjectMeta.Namespace)

	obj, err := gatewayClient.Get(gw.Name, metav1.GetOptions{})
	if err != nil {
		obj = gw.DeepCopy()
	}

	obj.Status = *gw.Status.DeepCopy()

	if err = wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
		g, err := gatewayClient.Update(obj)
		if err != nil {
			if !common.IsRetryableKubeAPIError(err) {
				return false, err
			}
			return false, nil
		}
		obj = g.DeepCopy()
		return true, nil
	}); err != nil {
		log.Warn("error updating the gateway", zap.Error(err))
		return nil, err
	}
	return obj, nil
}
