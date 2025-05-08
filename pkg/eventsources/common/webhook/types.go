/*
Copyright 2018 The Argoproj Authors.

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

package webhook

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
)

var (
	// Mutex synchronizes ActiveServerHandlers
	Lock sync.Mutex
)

// Router is an interface to manage the route
type Router interface {
	// GetRoute returns the route
	GetRoute() *Route
	// HandleRoute processes the incoming requests on the route
	HandleRoute(writer http.ResponseWriter, request *http.Request)
	// PostActivate captures the operations if any after route being activated and ready to process requests.
	PostActivate() error
	// PostInactivate captures cleanup operations if any after route is inactivated
	PostInactivate() error
}

// Dispatch is sent by RouteHandler function through
// the Route's DispatchChan and is used to coordinate writing to the
// event bus
type Dispatch struct {
	// Data contains the webhook data to dispatch to the event bus
	Data []byte
	// SuccessChan contains true iff the dispatch of the Data was successful
	SuccessChan chan bool
}

// Route contains general information about a route
type Route struct {
	// WebhookContext refers to the webhook context
	Context *aev1.WebhookContext
	// Logger to log stuff
	Logger *zap.SugaredLogger
	// StartCh controls the
	StartCh chan struct{}
	// EventSourceName refers to event source name
	EventSourceName string
	// EventName refers to event name
	EventName string
	// active determines whether the route is active and ready to process incoming requests
	// or it is an inactive route
	Active bool
	// data channel to receive data on this endpoint
	DispatchChan chan *Dispatch
	// Stop channel to signal the end of the event source.
	StopChan chan struct{}

	Metrics *metrics.Metrics
}

// Controller controls the active servers and endpoints
type Controller struct {
	// ActiveServerHandlers keeps track of currently active mux/router for the http servers.
	ActiveServerHandlers map[string]*mux.Router
	// AllRoutes keep track of routes that are already registered with server and their status active or inactive
	AllRoutes map[string]*mux.Route
	// RouteActivateChan handles activation of routes
	RouteActivateChan chan Router
	// RouteDeactivateChan handles inactivation of routes
	RouteDeactivateChan chan Router
}
