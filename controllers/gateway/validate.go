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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
)

// Validate validates the gateway resource.
// Exporting this function so that external APIs can use this to validate gateway resource.
func Validate(gw *v1alpha1.Gateway) error {
	if gw.Spec.DeploySpec == nil {
		return fmt.Errorf("gateway deploy specification is not specified")
	}
	if gw.Spec.Type == "" {
		return fmt.Errorf("gateway type is not specified")
	}
	if gw.Spec.EventVersion == "" {
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
