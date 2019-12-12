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

import "github.com/argoproj/argo-events/pkg/apis/gateway"

// Labels
const (
	//LabelKeyGatewayControllerInstanceID is the label which allows to separate application among multiple running controller controllers.
	LabelControllerInstanceID = gateway.FullName + "/gateway-controller-instanceid"
	// LabelGatewayKeyPhase is a label applied to gateways to indicate the current phase of the controller (for filtering purposes)
	LabelPhase = gateway.FullName + "/phase"
)
