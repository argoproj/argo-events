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
	return nil
}
