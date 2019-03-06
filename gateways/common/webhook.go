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
	// ServerCertPath refers the file that contains the cert.
	ServerCertPath string `json:"serverCertPath,omitempty" protobuf:"bytes,4,opt,name=serverCertPath"`
	// ServerKeyPath refers the file that contains private key
	ServerKeyPath string `json:"serverKeyPath,omitempty" protobuf:"bytes,5,opt,name=serverKeyPath"`
	// srv holds reference to http server
	srv *http.Server   `json:"srv,omitempty"`
	mux *http.ServeMux `json:"mux,omitempty"`
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
	RouteActivateChan chan *RouteConfig
	// RouteDeactivateChan handles deactivating existing route
	RouteDeactivateChan chan *RouteConfig
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

// RouteConfig contains configuration about a http route
type RouteConfig struct {
	Webhook            *Webhook
	Configs            map[string]interface{}
	EventSource        *gateways.EventSource
	Log                zerolog.Logger
	StartCh            chan struct{}
	RouteActiveHandler func(writer http.ResponseWriter, request *http.Request, rc *RouteConfig)
	PostActivate       func(rc *RouteConfig) error
	PostStop           func(rc *RouteConfig) error
}

// endpoint contains state of an http endpoint
type Endpoint struct {
	// whether endpoint is active
	Active bool
	// data channel to receive data on this endpoint
	DataCh chan []byte
}

// NewWebhookHelper returns new webhook helper
func NewWebhookHelper() *WebhookHelper {
	return &WebhookHelper{
		ActiveEndpoints:     make(map[string]*Endpoint),
		ActiveServers:       make(map[string]*activeServer),
		Mutex:               sync.Mutex{},
		RouteActivateChan:   make(chan *RouteConfig),
		RouteDeactivateChan: make(chan *RouteConfig),
	}
}

// InitRouteChannels initializes route channels so they can activate and deactivate routes.
func InitRouteChannels(helper *WebhookHelper) {
	for {
		select {
		case config := <-helper.RouteActivateChan:
			// start server if it has not been started on this port
			config.startHttpServer(helper)
			config.StartCh <- struct{}{}

		case config := <-helper.RouteDeactivateChan:
			webhook := config.Webhook
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
func (rc *RouteConfig) startHttpServer(helper *WebhookHelper) {
	// start a http server only if no other configuration previously started the server on given port
	helper.Mutex.Lock()
	if _, ok := helper.ActiveServers[rc.Webhook.Port]; !ok {
		s := &server{
			mux: http.NewServeMux(),
		}
		rc.Webhook.mux = s.mux
		rc.Webhook.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", rc.Webhook.Port),
			Handler: s,
		}
		errChan := make(chan error, 1)
		helper.ActiveServers[rc.Webhook.Port] = &activeServer{
			srv:     s.mux,
			errChan: errChan,
		}

		// start http server
		go func() {
			var err error
			if rc.Webhook.ServerCertPath == "" || rc.Webhook.ServerKeyPath == "" {
				err = rc.Webhook.srv.ListenAndServe()
			} else {
				err = rc.Webhook.srv.ListenAndServeTLS(rc.Webhook.ServerCertPath, rc.Webhook.ServerKeyPath)
			}
			rc.Log.Info().Str("event-source", rc.EventSource.Name).Str("port", rc.Webhook.Port).Msg("http server stopped")
			if err != nil {
				errChan <- err
			}
		}()
	}
	helper.Mutex.Unlock()
}

// activateRoute activates route
func (rc *RouteConfig) activateRoute(helper *WebhookHelper) {
	helper.RouteActivateChan <- rc

	<-rc.StartCh

	if rc.Webhook.mux == nil {
		helper.Mutex.Lock()
		rc.Webhook.mux = helper.ActiveServers[rc.Webhook.Port].srv
		helper.Mutex.Unlock()
	}

	rc.Log.Info().Str("event-source-name", rc.EventSource.Name).Str("port", rc.Webhook.Port).Str("endpoint", rc.Webhook.Endpoint).Msg("adding route handler")
	if _, ok := helper.ActiveEndpoints[rc.Webhook.Endpoint]; !ok {
		helper.ActiveEndpoints[rc.Webhook.Endpoint] = &Endpoint{
			Active: true,
			DataCh: make(chan []byte),
		}
		rc.Webhook.mux.HandleFunc(rc.Webhook.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
			rc.RouteActiveHandler(writer, request, rc)
		})
	}
	helper.ActiveEndpoints[rc.Webhook.Endpoint].Active = true

	rc.Log.Info().Str("event-source-name", rc.EventSource.Name).Str("port", rc.Webhook.Port).Str("endpoint", rc.Webhook.Endpoint).Msg("route handler added")
}

func (rc *RouteConfig) processChannels(helper *WebhookHelper, eventStream gateways.Eventing_StartEventSourceServer) error {
	for {
		select {
		case data := <-helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh:
			rc.Log.Info().Str("event-source-name", rc.EventSource.Name).Msg("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    rc.EventSource.Name,
				Payload: data,
			})
			if err != nil {
				rc.Log.Error().Err(err).Str("event-source-name", rc.EventSource.Name).Msg("failed to send event")
				return err
			}

		case <-eventStream.Context().Done():
			rc.Log.Info().Str("event-source-name", rc.EventSource.Name).Msg("connection is closed by client")
			helper.RouteDeactivateChan <- rc
			return nil

		// this error indicates that the server has stopped running
		case err := <-helper.ActiveServers[rc.Webhook.Port].errChan:
			return err
		}
	}
}

func DefaultPostActivate(rc *RouteConfig) error {
	return nil
}

func DefaultPostStop(rc *RouteConfig) error {
	return nil
}

func ProcessRoute(rc *RouteConfig, helper *WebhookHelper, eventStream gateways.Eventing_StartEventSourceServer) error {
	rc.activateRoute(helper)
	if err := rc.PostActivate(rc); err != nil {
		return err
	}
	if err := rc.processChannels(helper, eventStream); err != nil {
		return err
	}
	if err := rc.PostStop(rc); err != nil {
		rc.Log.Error().Err(err).Msg("error occurred while executing post stop logic")
	}
	return nil
}

func ValidateWebhook(endpoint, port string) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint can't be empty")
	}
	if port == "" {
		return fmt.Errorf("port can't be empty")
	}
	if port != "" {
		_, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("failed to parse server port %s. err: %+v", port, err)
		}
	}
	return nil
}

func FormatWebhookEndpoint(endpoint string) string {
	if !strings.HasPrefix(endpoint, "/") {
		return fmt.Sprintf("/%s", endpoint)
	}
	return endpoint
}
