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
	gwclient "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// the context of an operation in the controller.
// the controller creates this context each time it picks a Gateway off its queue.
type operationContext struct {
	// gatewayObj is the controller object
	gatewayObj *v1alpha1.Gateway
	// updated indicates whether the controller object was updated and needs to be persisted back to k8
	updated bool
	// logger is the logger for a gateway
	logger *logrus.Logger
	// reference to the controller
	controller *Controller
}

// newOperationCtx creates and initializes a new operationContext object
func newOperationCtx(gatewayObj *v1alpha1.Gateway, controller *Controller) *operationContext {
	gatewayObj = gatewayObj.DeepCopy()
	return &operationContext{
		gatewayObj: gatewayObj,
		updated:    false,
		logger: common.NewArgoEventsLogger().WithFields(
			map[string]interface{}{
				common.LabelResourceName: gatewayObj.Name,
				common.LabelNamespace:    gatewayObj.Namespace,
			}).Logger,
		controller: controller,
	}
}

// operate checks the status of gateway resource and takes action based on it.
func (opctx *operationContext) operate() error {
	defer opctx.updateGatewayState()

	opctx.logger.WithField(common.LabelPhase, string(opctx.gatewayObj.Status.Phase)).Infoln("operating on the gateway...")

	// check the state of a gateway and take actions accordingly
	switch opctx.gatewayObj.Status.Phase {
	case v1alpha1.NodePhaseNew:
		if err := opctx.createGatewayResources(); err != nil {
			opctx.logger.WithError(err).Errorln("failed to create resources for gateway object")
			opctx.markGatewayPhase(v1alpha1.NodePhaseError, err.Error())
			return err
		}

		opctx.logger.Infoln("marking gateway as active")

		opctx.markGatewayPhase(v1alpha1.NodePhaseRunning, "gateway is active")

	// Gateway is in error
	case v1alpha1.NodePhaseError:
		opctx.logger.Errorln("gateway is in error state. please check the escalated K8 event for any error")
		err := opctx.updateGatewayResources()
		if err != nil {
			return err
		}

	// Gateway is already running
	case v1alpha1.NodePhaseRunning:
		opctx.logger.Infoln("gateway is running")
		err := opctx.updateGatewayResources()
		if err != nil {
			return err
		}

	default:
		opctx.logger.WithField(common.LabelPhase, string(opctx.gatewayObj.Status.Phase)).Errorln("unknown gateway phase")
	}
	return nil
}

// updateGatewayState updates the gateway state
func (opctx *operationContext) updateGatewayState() {
	if opctx.updated {
		var err error
		eventType := common.StateChangeEventType
		labels := map[string]string{
			common.LabelResourceName:  opctx.gatewayObj.Name,
			LabelPhase:                string(opctx.gatewayObj.Status.Phase),
			LabelControllerInstanceID: opctx.controller.Config.InstanceID,
			common.LabelOperation:     "persist_gateway_state",
		}

		opctx.gatewayObj, err = PersistUpdates(opctx.controller.gatewayClient, opctx.gatewayObj, opctx.logger)
		if err != nil {
			opctx.logger.WithError(err).Errorln("failed to persist gateway update, escalating...")
			eventType = common.EscalationEventType
		}

		labels[common.LabelEventType] = string(eventType)
		if err := common.GenerateK8sEvent(opctx.controller.k8sClient,
			"persist update",
			eventType,
			"gateway state update",
			opctx.gatewayObj.Name,
			opctx.gatewayObj.Namespace,
			opctx.controller.Config.InstanceID,
			gateway.Kind,
			labels,
		); err != nil {
			opctx.logger.WithError(err).Errorln("failed to create K8s event to logger gateway state persist operation")
			return
		}
		opctx.logger.Infoln("successfully persisted gateway resource update and created K8s event")
	}
	opctx.updated = false
}

// mark the gateway phase
func (opctx *operationContext) markGatewayPhase(phase v1alpha1.NodePhase, message string) {
	justCompleted := opctx.gatewayObj.Status.Phase != phase
	if justCompleted {
		opctx.logger.WithFields(
			map[string]interface{}{
				"old": string(opctx.gatewayObj.Status.Phase),
				"new": string(phase),
			},
		).Infoln("phase changed")

		opctx.gatewayObj.Status.Phase = phase
		if opctx.gatewayObj.ObjectMeta.Labels == nil {
			opctx.gatewayObj.ObjectMeta.Labels = make(map[string]string)
		}
		if opctx.gatewayObj.Annotations == nil {
			opctx.gatewayObj.Annotations = make(map[string]string)
		}

		opctx.gatewayObj.ObjectMeta.Labels[LabelPhase] = string(phase)
		// add annotations so a resource sensor can watch this gateway.
		opctx.gatewayObj.Annotations[LabelPhase] = string(phase)
	}

	if opctx.gatewayObj.Status.StartedAt.IsZero() {
		opctx.gatewayObj.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
	}

	opctx.logger.WithFields(
		map[string]interface{}{
			"old": string(opctx.gatewayObj.Status.Message),
			"new": message,
		},
	).Infoln("phase change message")

	opctx.gatewayObj.Status.Message = message
	opctx.updated = true
}

// PersistUpdates of the gateway resource
func PersistUpdates(client gwclient.Interface, gw *v1alpha1.Gateway, log *logrus.Logger) (*v1alpha1.Gateway, error) {
	gatewayClient := client.ArgoprojV1alpha1().Gateways(gw.ObjectMeta.Namespace)

	// in case persist update fails
	oldgw := gw.DeepCopy()

	gw, err := gatewayClient.Update(gw)
	if err != nil {
		log.WithError(err).Warn("error updating gateway")
		if errors.IsConflict(err) {
			return oldgw, err
		}
		log.Info("re-applying updates on latest version and retrying update")
		err = ReapplyUpdates(client, gw)
		if err != nil {
			log.WithError(err).Error("failed to re-apply update")
			return oldgw, err
		}
	}
	log.WithField(common.LabelPhase, string(gw.Status.Phase)).Info("gateway state updated successfully")
	return gw, nil
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
