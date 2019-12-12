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
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/pkg/errors"
)

// Validate validates the gateway resource.
func Validate(gatewayObj *v1alpha1.Gateway) error {
	if gatewayObj.Spec.Template == nil {
		return errors.New("gateway  pod template is not specified")
	}
	if gatewayObj.Spec.Type == "" {
		return errors.New("gateway type is not specified")
	}
	if gatewayObj.Spec.EventSourceRef == nil {
		return errors.New("event source for the gateway is not specified")
	}

	if gatewayObj.Spec.ProcessorPort == "" {
		return errors.New("gateway processor port is not specified")
	}

	switch gatewayObj.Spec.EventProtocol.Type {
	case apicommon.HTTP:
		if gatewayObj.Spec.Watchers == nil || (gatewayObj.Spec.Watchers.Gateways == nil && gatewayObj.Spec.Watchers.Sensors == nil) {
			return errors.New("no associated watchers with gateway")
		}
		if gatewayObj.Spec.EventProtocol.Http.Port == "" {
			return errors.New("http server port is not defined")
		}
	case apicommon.NATS:
		if gatewayObj.Spec.EventProtocol.Nats.URL == "" {
			return errors.New("nats url is not defined")
		}
		if gatewayObj.Spec.EventProtocol.Nats.Type == "" {
			return errors.New("nats service type is not defined")
		}
		if gatewayObj.Spec.EventProtocol.Nats.Type == apicommon.Streaming && gatewayObj.Spec.EventProtocol.Nats.ClientId == "" {
			return errors.New("client id must be specified when using nats streaming")
		}
		if gatewayObj.Spec.EventProtocol.Nats.Type == apicommon.Streaming && gatewayObj.Spec.EventProtocol.Nats.ClusterId == "" {
			return errors.New("cluster id must be specified when using nats streaming")
		}
	default:
		return errors.New("unknown gateway type")
	}
	return nil
}
