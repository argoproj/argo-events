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

package trello

import (
	"github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type TrelloEventSourceExecutor struct {
	Log zerolog.Logger
	// Clientset is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

type trello struct {
	// Webhook
	Hook *common.Webhook `json:"hook"`
	// ApiKey for client
	ApiKey *corev1.SecretKeySelector `json:"apiKey"`
	// Token for client
	Token *corev1.SecretKeySelector `json:"token"`
	// Description for webhook
	// +optional
	Description string `json:"description,omitempty"`
}

func parseEventSource(es string) (interface{}, error) {
	var n *trello
	err := yaml.Unmarshal([]byte(es), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
