package gateway

import "fmt"

// Validates the gateway resource
func (goc *gwOperationCtx) validate() error {
	if goc.gw.Spec.DeploySpec == nil {
		return fmt.Errorf("gateway deploy specification is not specified")
	}
	if goc.gw.Spec.Type == "" {
		return fmt.Errorf("gateway type is not specified")
	}
	if goc.gw.Spec.Version == "" {
		return fmt.Errorf("gateway version is not specified")
	}
	if len(goc.gw.Spec.Sensors) <= 0 {
		return fmt.Errorf("no associated sensor with gateway")
	}
	return nil
}
