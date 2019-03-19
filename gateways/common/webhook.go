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

	"github.com/argoproj/argo-events/gateways"
	"github.com/rs/zerolog"
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
	RouteActivateChan chan RouteConfig
	// RouteDeactivateChan handles deactivating existing route
	RouteDeactivateChan chan RouteConfig
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
	Logger      *zerolog.Logger
	StartCh     chan struct{}
	EventSource *gateways.EventSource
}

// RouteConfig is an interface to manage  the configuration for a route
type RouteConfig interface {
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

// NewWebhookHelper returns new r.Webhook helper
func NewWebhookHelper() *WebhookHelper {
	return &WebhookHelper{
		ActiveEndpoints:     make(map[string]*Endpoint),
		ActiveServers:       make(map[string]*activeServer),
		Mutex:               sync.Mutex{},
		RouteActivateChan:   make(chan RouteConfig),
		RouteDeactivateChan: make(chan RouteConfig),
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
func startHttpServer(rc RouteConfig, helper *WebhookHelper) {
	// start a http server only if no other configuration previously started the server on given port
	helper.Mutex.Lock()
	r := rc.GetRoute()
	if _, ok := helper.ActiveServers[r.Webhook.Port]; !ok {
		s := &server{
			mux: http.NewServeMux(),
		}
		r.Webhook.mux = s.mux
		r.Webhook.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", r.Webhook.Port),
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
			r.Logger.Error().Err(err).Str("event-source", r.EventSource.Name).Str("port", r.Webhook.Port).Msg("http server stopped")
			if err != nil {
				errChan <- err
			}
		}()
	}
	helper.Mutex.Unlock()
}

// activateRoute activates route
func activateRoute(rc RouteConfig, helper *WebhookHelper) {
	r := rc.GetRoute()
	helper.RouteActivateChan <- rc

	<-r.StartCh

	if r.Webhook.mux == nil {
		helper.Mutex.Lock()
		r.Webhook.mux = helper.ActiveServers[r.Webhook.Port].srv
		helper.Mutex.Unlock()
	}

	r.Logger.Info().Str("event-source-name", r.EventSource.Name).Str("port", r.Webhook.Port).Str("endpoint", r.Webhook.Endpoint).Msg("adding route handler")
	if _, ok := helper.ActiveEndpoints[r.Webhook.Endpoint]; !ok {
		helper.ActiveEndpoints[r.Webhook.Endpoint] = &Endpoint{
			Active: true,
			DataCh: make(chan []byte),
		}
		r.Webhook.mux.HandleFunc(r.Webhook.Endpoint, rc.RouteHandler)
	}
	helper.ActiveEndpoints[r.Webhook.Endpoint].Active = true

	r.Logger.Info().Str("event-source-name", r.EventSource.Name).Str("port", r.Webhook.Port).Str("endpoint", r.Webhook.Endpoint).Msg("route handler added")
}

func processChannels(rc RouteConfig, helper *WebhookHelper, eventStream gateways.Eventing_StartEventSourceServer) error {
	r := rc.GetRoute()

	for {
		select {
		case data := <-helper.ActiveEndpoints[r.Webhook.Endpoint].DataCh:
			r.Logger.Info().Str("event-source-name", r.EventSource.Name).Msg("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    r.EventSource.Name,
				Payload: data,
			})
			if err != nil {
				r.Logger.Error().Err(err).Str("event-source-name", r.EventSource.Name).Msg("failed to send event")
				return err
			}

		case <-eventStream.Context().Done():
			r.Logger.Info().Str("event-source-name", r.EventSource.Name).Msg("connection is closed by client")
			helper.RouteDeactivateChan <- rc
			return nil

		// this error indicates that the server has stopped running
		case err := <-helper.ActiveServers[r.Webhook.Port].errChan:
			return err
		}
	}
}

func ProcessRoute(rc RouteConfig, helper *WebhookHelper, eventStream gateways.Eventing_StartEventSourceServer) error {
	r := rc.GetRoute()

	if err := validateRoute(rc.GetRoute()); err != nil {
		r.Logger.Error().Err(err).Str("event-source", r.EventSource.Name).Msg("error occurred validating route")
		return err
	}

	activateRoute(rc, helper)

	if err := rc.PostStart(); err != nil {
		r.Logger.Error().Err(err).Str("event-source", r.EventSource.Name).Msg("error occurred in post start")
		return err
	}

	if err := processChannels(rc, helper, eventStream); err != nil {
		r.Logger.Error().Err(err).Str("event-source", r.EventSource.Name).Msg("error occurred in process channel")
		return err
	}

	if err := rc.PostStop(); err != nil {
		r.Logger.Error().Err(err).Str("event-source", r.EventSource.Name).Msg("error occurred in post stop")
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
