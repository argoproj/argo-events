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
	"github.com/argoproj/argo-events/pkg/common"
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
	switch gw.Spec.EventProtocol.Type {
	case common.HTTP:
		if gw.Spec.Watchers == nil || (gw.Spec.Watchers.Gateways == nil && gw.Spec.Watchers.Sensors == nil) {
			return fmt.Errorf("no associated watchers with gateway")
		}
		if gw.Spec.EventProtocol.Http.Port == "" {
			return fmt.Errorf("http server port is not defined")
		}
	case common.NATS:
		if gw.Spec.EventProtocol.Nats.URL == "" {
			return fmt.Errorf("nats url is not defined")
		}
		if gw.Spec.EventProtocol.Nats.Type == "" {
			return fmt.Errorf("nats service type is not defined")
		}
		if gw.Spec.EventProtocol.Nats.Type == common.Streaming && gw.Spec.EventProtocol.Nats.ClientId == "" {
			return fmt.Errorf("client id must be specified when using nats streaming")
		}
		if gw.Spec.EventProtocol.Nats.Type == common.Streaming && gw.Spec.EventProtocol.Nats.ClusterId == "" {
			return fmt.Errorf("cluster id must be specified when using nats streaming")
		}
	default:
		return fmt.Errorf("unknown gateway type")
	}
	return nil
}
