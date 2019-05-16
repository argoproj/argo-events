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

package common

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/sirupsen/logrus"
)

// Webhook is a general purpose REST API
type Webhook struct {
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

	// srv holds reference to http server
	srv *http.Server
	mux *http.ServeMux
}

// WebhookHelper is a helper struct
type WebhookHelper struct {
	// Mutex synchronizes ActiveServers
	Mutex sync.Mutex
	// ActiveServers keeps track of currently running http servers.
	ActiveServers map[string]*activeServer
	// ActiveEndpoints keep track of endpoints that are already registered with server and their status active or inactive
	ActiveEndpoints map[string]*Endpoint
	// RouteActivateChan handles assigning new route to server.
	RouteActivateChan chan RouteManager
	// RouteDeactivateChan handles deactivating existing route
	RouteDeactivateChan chan RouteManager
}

// HTTP Muxer
type server struct {
	mux *http.ServeMux
}

// activeServer contains reference to server and an error channel that is shared across all functions registering endpoints for the server.
type activeServer struct {
	srv     *http.ServeMux
	errChan chan error
}

// Route contains common information for a route
type Route struct {
	Webhook     *Webhook
	Logger      *logrus.Logger
	StartCh     chan struct{}
	EventSource *gateways.EventSource
}

// RouteManager is an interface to manage the configuration for a route
type RouteManager interface {
	GetRoute() *Route
	RouteHandler(writer http.ResponseWriter, request *http.Request)
	PostStart() error
	PostStop() error
}

// endpoint contains state of an http endpoint
type Endpoint struct {
	// whether endpoint is active
	Active bool
	// data channel to receive data on this endpoint
	DataCh chan []byte
}

// NewWebhookHelper returns new Webhook helper
func NewWebhookHelper() *WebhookHelper {
	return &WebhookHelper{
		ActiveEndpoints:     make(map[string]*Endpoint),
		ActiveServers:       make(map[string]*activeServer),
		Mutex:               sync.Mutex{},
		RouteActivateChan:   make(chan RouteManager),
		RouteDeactivateChan: make(chan RouteManager),
	}
}

// InitRouteChannels initializes route channels so they can activate and deactivate routes.
func InitRouteChannels(helper *WebhookHelper) {
	for {
		select {
		case config := <-helper.RouteActivateChan:
			// start server if it has not been started on this port
			startHttpServer(config, helper)
			startCh := config.GetRoute().StartCh
			startCh <- struct{}{}

		case config := <-helper.RouteDeactivateChan:
			webhook := config.GetRoute().Webhook
			_, ok := helper.ActiveServers[webhook.Port]
			if ok {
				helper.ActiveEndpoints[webhook.Endpoint].Active = false
			}
		}
	}
}

// ServeHTTP implementation
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// starts a http server
func startHttpServer(routeManager RouteManager, helper *WebhookHelper) {
	// start a http server only if no other configuration previously started the server on given port
	helper.Mutex.Lock()
	r := routeManager.GetRoute()
	if _, ok := helper.ActiveServers[r.Webhook.Port]; !ok {
		s := &server{
			mux: http.NewServeMux(),
		}
		r.Webhook.mux = s.mux
		r.Webhook.srv = &http.Server{
			Addr:    ":" + r.Webhook.Port,
			Handler: s,
		}
		errChan := make(chan error, 1)
		helper.ActiveServers[r.Webhook.Port] = &activeServer{
			srv:     s.mux,
			errChan: errChan,
		}

		// start http server
		go func() {
			var err error
			if r.Webhook.ServerCertPath == "" || r.Webhook.ServerKeyPath == "" {
				err = r.Webhook.srv.ListenAndServe()
			} else {
				err = r.Webhook.srv.ListenAndServeTLS(r.Webhook.ServerCertPath, r.Webhook.ServerKeyPath)
			}
			r.Logger.WithField(common.LabelEventSource, r.EventSource.Name).WithError(err).Error("http server stopped")
			if err != nil {
				errChan <- err
			}
		}()
	}
	helper.Mutex.Unlock()
}

// activateRoute activates route
func activateRoute(routeManager RouteManager, helper *WebhookHelper) {
	r := routeManager.GetRoute()
	helper.RouteActivateChan <- routeManager

	<-r.StartCh

	if r.Webhook.mux == nil {
		helper.Mutex.Lock()
		r.Webhook.mux = helper.ActiveServers[r.Webhook.Port].srv
		helper.Mutex.Unlock()
	}

	log := r.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: r.EventSource.Name,
			common.LabelPort:        r.Webhook.Port,
			common.LabelEndpoint:    r.Webhook.Endpoint,
		})

	log.Info("adding route handler")
	if _, ok := helper.ActiveEndpoints[r.Webhook.Endpoint]; !ok {
		helper.ActiveEndpoints[r.Webhook.Endpoint] = &Endpoint{
			Active: true,
			DataCh: make(chan []byte),
		}
		r.Webhook.mux.HandleFunc(r.Webhook.Endpoint, routeManager.RouteHandler)
	}
	helper.ActiveEndpoints[r.Webhook.Endpoint].Active = true

	log.Info("route handler added")
}

func processChannels(routeManager RouteManager, helper *WebhookHelper, eventStream gateways.Eventing_StartEventSourceServer) error {
	r := routeManager.GetRoute()

	for {
		select {
		case data := <-helper.ActiveEndpoints[r.Webhook.Endpoint].DataCh:
			r.Logger.WithField(common.LabelEventSource, r.EventSource.Name).Info("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    r.EventSource.Name,
				Payload: data,
			})
			if err != nil {
				r.Logger.WithField(common.LabelEventSource, r.EventSource.Name).WithError(err).Error("failed to send event")
				return err
			}

		case <-eventStream.Context().Done():
			r.Logger.WithField(common.LabelEventSource, r.EventSource.Name).Info("connection is closed by client")
			helper.RouteDeactivateChan <- routeManager
			return nil

		// this error indicates that the server has stopped running
		case err := <-helper.ActiveServers[r.Webhook.Port].errChan:
			return err
		}
	}
}

func ProcessRoute(routeManager RouteManager, helper *WebhookHelper, eventStream gateways.Eventing_StartEventSourceServer) error {
	r := routeManager.GetRoute()
	log := r.Logger.WithField(common.LabelEventSource, r.EventSource.Name)

	log.Info("validating the route")
	if err := validateRoute(routeManager.GetRoute()); err != nil {
		log.WithError(err).Error("error occurred validating route")
		return err
	}

	log.Info("activating the route")
	activateRoute(routeManager, helper)

	log.Info("running post start")
	if err := routeManager.PostStart(); err != nil {
		log.WithError(err).Error("error occurred in post start")
		return err
	}

	log.Info("processing channels")
	if err := processChannels(routeManager, helper, eventStream); err != nil {
		log.WithError(err).Error("error occurred in process channel")
		return err
	}

	log.Info("running post stop")
	if err := routeManager.PostStop(); err != nil {
		log.WithError(err).Error("error occurred in post stop")
	}
	return nil
}

func ValidateWebhook(w *Webhook) error {
	if w == nil {
		return fmt.Errorf("")
	}
	if w.Endpoint == "" {
		return fmt.Errorf("endpoint can't be empty")
	}
	if w.Port == "" {
		return fmt.Errorf("port can't be empty")
	}
	if w.Port != "" {
		_, err := strconv.Atoi(w.Port)
		if err != nil {
			return fmt.Errorf("failed to parse server port %s. err: %+v", w.Port, err)
		}
	}
	return nil
}

func validateRoute(r *Route) error {
	if r == nil {
		return fmt.Errorf("route can't be nil")
	}
	if r.Webhook == nil {
		return fmt.Errorf("webhook can't be nil")
	}
	if r.StartCh == nil {
		return fmt.Errorf("start channel can't be nil")
	}
	if r.EventSource == nil {
		return fmt.Errorf("event source can't be nil")
	}
	if r.Logger == nil {
		return fmt.Errorf("logger can't be nil")
	}
	return nil
}

func FormatWebhookEndpoint(endpoint string) string {
	if !strings.HasPrefix(endpoint, "/") {
		return fmt.Sprintf("/%s", endpoint)
	}
	return endpoint
}

func GenerateFormattedURL(w *Webhook) string {
	return fmt.Sprintf("%s%s", w.URL, FormatWebhookEndpoint(w.Endpoint))
}
