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
	"fmt"
	"github.com/argoproj/argo-events/common"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/argoproj/argo-events/gateways"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers.
	activeServers = make(map[string]*activeServer)

	// activeEndpoints keep track of endpoints that are already registered with server and their status active or deactive
	activeEndpoints = make(map[string]*endpoint)

	// routeActivateChan handles assigning new route to server.
	routeActivateChan = make(chan *routeConfig)

	// routeDeactivateChan handles deactivating existing route
	routeDeactivateChan = make(chan *routeConfig)
)

// HTTP Muxer
type server struct {
	mux *http.ServeMux
}

// activeServer contains reference to server and an error channel that is shared across all functions registering endpoints for the server.
type activeServer struct {
	srv     *http.ServeMux
	errChan chan error
}

type routeConfig struct {
	wConfig             *webhook
	eventSource         *gateways.EventSource
	eventSourceExecutor *WebhookEventSourceExecutor
	startCh             chan struct{}
}

type endpoint struct {
	active bool
	dataCh chan []byte
}

func init() {
	go func() {
		for {
			select {
			case config := <-routeActivateChan:
				// start server if it has not been started on this port
				config.startHttpServer()
				config.startCh <- struct{}{}

			case config := <-routeDeactivateChan:
				_, ok := activeServers[config.wConfig.Port]
				if ok {
					activeEndpoints[config.wConfig.Endpoint].active = false
				}
			}
		}
	}()
}

// ServeHTTP implementation
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// starts a http server
func (rc *routeConfig) startHttpServer() {
	// start a http server only if no other configuration previously started the server on given port
	mutex.Lock()
	if _, ok := activeServers[rc.wConfig.Port]; !ok {
		s := &server{
			mux: http.NewServeMux(),
		}
		rc.wConfig.mux = s.mux
		rc.wConfig.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", rc.wConfig.Port),
			Handler: s,
		}
		errChan := make(chan error, 1)
		activeServers[rc.wConfig.Port] = &activeServer{
			srv:     s.mux,
			errChan: errChan,
		}

		// start http server
		go func() {
			err := rc.wConfig.srv.ListenAndServe()
			rc.eventSourceExecutor.Log.Info().Str("event-source", rc.eventSource.Name).Str("port", rc.wConfig.Port).Msg("http server stopped")
			if err != nil {
				errChan <- err
			}
		}()
	}
	mutex.Unlock()
}

// routeActiveHandler handles new route
func (rc *routeConfig) routeActiveHandler(writer http.ResponseWriter, request *http.Request) {
	var response string
	if !activeEndpoints[rc.wConfig.Endpoint].active {
		response = fmt.Sprintf("the route: endpoint %s and method %s is deactived", rc.wConfig.Endpoint, rc.wConfig.Method)
		rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.wConfig.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("endpoint is not active")
		common.SendErrorResponse(writer, response)
		return
	}
	if rc.wConfig.Method != request.Method {
		msg := fmt.Sprintf("the method %s is not defined for endpoint %s", rc.wConfig.Method, rc.wConfig.Endpoint)
		rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.wConfig.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("endpoint is not active")
		common.SendErrorResponse(writer, msg)
		return
	}

	rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.wConfig.Endpoint).Str("http-method", request.Method).Msg("payload received")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rc.eventSourceExecutor.Log.Error().Err(err).Str("endpoint", rc.wConfig.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("failed to parse request body")
		common.SendErrorResponse(writer, fmt.Sprintf("failed to parse request. err: %+v", err))
		return
	}

	activeEndpoints[rc.wConfig.Endpoint].dataCh <- body
	response = "request successfully processed"
	rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.wConfig.Endpoint).Str("http-method", request.Method).Str("response", response).Msg("request payload parsed successfully")
	common.SendSuccessResponse(writer, response)
}

// StartEventSource starts a event source
func (ese *WebhookEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	h, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	rc := &routeConfig{
		wConfig:             h,
		eventSource:         eventSource,
		eventSourceExecutor: ese,
		startCh:             make(chan struct{}),
	}

	routeActivateChan <- rc

	<-rc.startCh

	if rc.wConfig.mux == nil {
		mutex.Lock()
		rc.wConfig.mux = activeServers[rc.wConfig.Port].srv
		mutex.Unlock()
	}

	ese.Log.Info().Str("event-source-name", eventSource.Name).Str("port", h.Port).Str("endpoint", h.Endpoint).Str("method", h.Method).Msg("adding route handler")
	if _, ok := activeEndpoints[rc.wConfig.Endpoint]; !ok {
		activeEndpoints[rc.wConfig.Endpoint] = &endpoint{
			active: true,
			dataCh: make(chan []byte),
		}
		rc.wConfig.mux.HandleFunc(rc.wConfig.Endpoint, rc.routeActiveHandler)
	}
	activeEndpoints[rc.wConfig.Endpoint].active = true

	ese.Log.Info().Str("event-source-name", eventSource.Name).Str("port", h.Port).Str("endpoint", h.Endpoint).Str("method", h.Method).Msg("route handler added")
	for {
		select {
		case data := <-activeEndpoints[rc.wConfig.Endpoint].dataCh:
			ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    eventSource.Name,
				Payload: data,
			})
			if err != nil {
				ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Msg("failed to send event")
				return err
			}

		case <-eventStream.Context().Done():
			ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("connection is closed by client")
			routeDeactivateChan <- rc
			return nil

		// this error indicates that the server has stopped running
		case err := <-activeServers[rc.wConfig.Port].errChan:
			return err
		}
	}
}
