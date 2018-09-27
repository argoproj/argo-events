package gateway

import (
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
)

// Validates the gateway resource.
// Exporting this function so that external APIs can use this to validate gateway resource.
func Validate(gw *v1alpha1.Gateway) error {
	if gw.Spec.DeploySpec == nil {
		return fmt.Errorf("gateway deploy specification is not specified")
	}
	if gw.Spec.Type == "" {
		return fmt.Errorf("gateway type is not specified")
	}
	if gw.Spec.Version == "" {
		return fmt.Errorf("gateway version is not specified")
	}
	switch gw.Spec.DispatchMechanism {
	case v1alpha1.HTTPGateway:
		if gw.Spec.Watchers == nil || (gw.Spec.Watchers.Gateways == nil && gw.Spec.Watchers.Sensors == nil) {
			return fmt.Errorf("no associated watchers with gateway")
		}
	case v1alpha1.NATSGateway:
	case v1alpha1.KafkaGateway:
	default:
		return fmt.Errorf("unknown gateway type")
	}
	return nil
}
