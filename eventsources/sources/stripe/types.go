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
	"github.com/argoproj/argo-events/eventsources/common/webhook"
	metrics "github.com/argoproj/argo-events/metrics"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventListener implements Eventing for stripe event source
type EventListener struct {
	EventSourceName   string
	EventName         string
	StripeEventSource v1alpha1.StripeEventSource
	Metrics           *metrics.Metrics
}

// Router contains information about a REST endpoint
type Router struct {
	// route holds information to process an incoming request
	route *webhook.Route
	// stripeEventSource is the event source which refers to configuration required to consume events from stripe
	stripeEventSource *v1alpha1.StripeEventSource
}
