package gateway

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwclient "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// persist the updates to the gateway resource
func PersistUpdates(client gwclient.Interface, gw *v1alpha1.Gateway, log *zerolog.Logger) (*v1alpha1.Gateway, error) {
	gatewayClient := client.ArgoprojV1alpha1().Gateways(gw.ObjectMeta.Namespace)

	// in case persist update fails
	oldgw := gw.DeepCopy()

	gw, err := gatewayClient.Update(gw)
	if err != nil {
		log.Warn().Err(err).Msg("error updating gateway")
		if errors.IsConflict(err) {
			return oldgw, err
		}
		log.Info().Msg("re-applying updates on latest version and retrying update")
		err = ReapplyUpdate(client, gw)
		if err != nil {
			log.Error().Err(err).Msg("failed to re-apply update")
			return oldgw, err
		}
	}
	log.Info().Str("gateway-phase", string(gw.Status.Phase)).Msg("gateway state updated successfully")
	return gw, nil
}

// reapply the updates to gateway resource
func ReapplyUpdate(client gwclient.Interface, gw *v1alpha1.Gateway) error {
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
