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

package stripe

import (
	"encoding/json"
	"github.com/argoproj/argo-events/gateways/server/common/webhook"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// EventListener implements Eventing for stripe event source
type EventListener struct {
	// K8sClient is kubernetes client
	K8sClient kubernetes.Interface
	// Logger logs stuff
	Logger *logrus.Logger
}

// Router contains information about a REST endpoint
type Router struct {
	// route holds information to process an incoming request
	route *webhook.Route
	// stripeEventSource is the event source which refers to configuration required to consume events from stripe
	stripeEventSource *v1alpha1.StripeEventSource
	// k8sClient is the Kubernetes client
	k8sClient kubernetes.Interface
}

// Response describes the event dispatched by the gateway
type Response struct {
	// Id of the stripe event
	Id string `json:"id"`
	// EventType refers to stripe event type
	EventType string `json:"eventType"`
	// Data refers to event data
	Data json.RawMessage `json:"data"`
}
