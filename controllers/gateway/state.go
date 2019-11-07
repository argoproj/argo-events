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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwclient "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

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
