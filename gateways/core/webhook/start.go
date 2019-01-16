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
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers.
	activeServers = make(map[string]*activeServer)

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
	dataCh              chan []byte
	doneCh              chan struct{}
	errCh               chan error
	startCh             chan struct{}
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
					config.wConfig.mux.HandleFunc(config.wConfig.Endpoint, config.routeDeactivateHandler)
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
	rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.wConfig.Endpoint).Str("http-method", request.Method).Msg("received a request")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rc.eventSourceExecutor.Log.Error().Err(err).Msg("failed to parse request body")
		rc.errCh <- err
		return
	}
	rc.dataCh <- body
}

// routeDeactivateHandler handles routes that are not active
func (rc *routeConfig) routeDeactivateHandler(writer http.ResponseWriter, request *http.Request) {
	rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.wConfig.Endpoint).Str("http-method", request.Method).Msg("route is not active")
	common.SendErrorResponse(writer)
}

// StartEventSource starts a event source
func (ese *WebhookEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	h, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	rc := &routeConfig{
		wConfig:             h,
		eventSource:         eventSource,
		eventSourceExecutor: ese,
		errCh:               make(chan error),
		dataCh:              make(chan []byte),
		doneCh:              make(chan struct{}),
		startCh:             make(chan struct{}),
	}

	routeActivateChan <- rc

	<-rc.startCh

	if rc.wConfig.mux == nil {
		mutex.Lock()
		rc.wConfig.mux = activeServers[rc.wConfig.Port].srv
		mutex.Unlock()
	}

	rc.wConfig.mux.HandleFunc(rc.wConfig.Endpoint, rc.routeActiveHandler)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Str("port", h.Port).Str("endpoint", h.Endpoint).Str("method", h.Method).Msg("route handler added")

	for {
		select {
		case data := <-rc.dataCh:
			ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("new event received, dispatching to gateway client")
			err := eventStream.Send(&gateways.Event{
				Name:    eventSource.Name,
				Payload: data,
			})
			if err != nil {
				return err
			}

		case err := <-rc.errCh:
			routeDeactivateChan <- rc
			return err

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
