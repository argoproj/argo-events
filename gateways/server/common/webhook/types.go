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

package webhook

import (
	"net/http"
	"sync"

	"github.com/argoproj/argo-events/gateways"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
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

// Route contains general information about a route
type Route struct {
	// Context refers to the webhook context
	Context *Context
	// Logger to log stuff
	Logger *logrus.Logger
	// StartCh controls the
	StartCh chan struct{}
	// EventSource refers to gateway event source
	EventSource *gateways.EventSource
	// active determines whether the route is active and ready to process incoming requets
	// or it is an inactive route
	Active bool
	// data channel to receive data on this endpoint
	DataCh chan []byte
	// Stop channel to signal the end of the event source.
	StopChan chan struct{}
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

// Context holds a general purpose REST API context
type Context struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,name=endpoint"`
	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,name=method"`
	// Port on which HTTP server is listening for incoming events.
	Port string `json:"port" protobuf:"bytes,3,name=port"`
	// URL is the url of the server.
	URL string `json:"url" protobuf:"bytes,4,name=url"`
	// ServerCertPath refers the file that contains the cert.
	ServerCertPath string `json:"serverCertPath,omitempty" protobuf:"bytes,4,opt,name=serverCertPath"`
	// ServerKeyPath refers the file that contains private key
	ServerKeyPath string `json:"serverKeyPath,omitempty" protobuf:"bytes,5,opt,name=serverKeyPath"`
}
