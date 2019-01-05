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

package storagegrid

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/joncalhoun/qson"
	"github.com/satori/go.uuid"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers.
	activeServers = make(map[string]*activeServer)

	// mutex synchronizes activeRoutes
	routesMutex sync.Mutex
	// activeRoutes keep track of active routes for a http server
	activeRoutes = make(map[string]map[string]struct{})

	// routeActivateChan handles assigning new route to server.
	routeActivateChan = make(chan routeConfig)

	routeDeactivateChan = make(chan routeConfig)

	respBody = `
<PublishResponse xmlns="http://argoevents-sns-server/">
    <PublishResult> 
        <MessageId>` + generateUUID().String() + `</MessageId> 
    </PublishResult> 
    <ResponseMetadata>
       <RequestId>` + generateUUID().String() + `</RequestId>
    </ResponseMetadata> 
</PublishResponse>` + "\n"
)

// HTTP Muxer
type server struct {
	mux *http.ServeMux
}

type routeConfig struct {
	sgConfig            *StorageGridEventConfig
	eventSource         *gateways.EventSource
	eventSourceExecutor *StorageGridEventSourceExecutor
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
				_, ok := activeServers[config.sgConfig.Port]
				if !ok {
					config.startHttpServer()
				}
				config.sgConfig.mux.HandleFunc(config.sgConfig.Endpoint, config.routeActiveHandler)

			case config := <-routeDeactivateChan:
				_, ok := activeServers[config.sgConfig.Port]
				if ok {
					config.sgConfig.mux.HandleFunc(config.sgConfig.Endpoint, config.routeDeactivateHandler)
				}
			}
		}
	}()
}

// activeServer contains reference to server and an error channel that is shared across all functions registering endpoints for the server.
type activeServer struct {
	srv     *http.ServeMux
	errChan chan error
}

// ServeHTTP implementation
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// generateUUID returns a new uuid
func generateUUID() uuid.UUID {
	return uuid.NewV4()
}

// filterEvent filters notification based on event filter in a gateway configuration
func filterEvent(notification *storageGridNotification, sg *StorageGridEventConfig) bool {
	if sg.Events == nil {
		return true
	}
	for _, filterEvent := range sg.Events {
		if notification.Message.Records[0].EventName == filterEvent {
			return true
		}
	}
	return false
}

// filterName filters object key based on configured prefix and/or suffix
func filterName(notification *storageGridNotification, sg *StorageGridEventConfig) bool {
	if sg.Filter == nil {
		return true
	}
	if sg.Filter.Prefix != "" && sg.Filter.Suffix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Prefix) && strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Suffix)
	}
	if sg.Filter.Prefix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Prefix)
	}
	if sg.Filter.Suffix != "" {
		return strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Suffix)
	}
	return true
}

// starts a http server
func (rc *routeConfig) startHttpServer() {
	// start a http server only if no other configuration previously started the server on given port
	mutex.Lock()
	if _, ok := activeServers[rc.sgConfig.Port]; !ok {
		s := &server{
			mux: http.NewServeMux(),
		}
		rc.sgConfig.mux = s.mux
		rc.sgConfig.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", rc.sgConfig.Port),
			Handler: s,
		}
		errChan := make(chan error, 1)
		activeServers[rc.sgConfig.Port] = &activeServer{
			srv:     s.mux,
			errChan: errChan,
		}

		// start http server
		go func() {
			err := rc.sgConfig.srv.ListenAndServe()
			rc.eventSourceExecutor.Log.Info().Str("event-source", *rc.eventSource.Name).Str("port", rc.sgConfig.Port).Msg("http server stopped")
			if err != nil {
				errChan <- err
			}
		}()
	}
	mutex.Unlock()
}

// StartConfig runs a configuration
func (ese *StorageGridEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", *eventSource.Name).Msg("operating on event source")
	sg, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	rc := routeConfig{
		sgConfig:            sg,
		eventSource:         eventSource,
		eventSourceExecutor: ese,
		errCh:               make(chan error),
		dataCh:              make(chan []byte),
		doneCh:              make(chan struct{}),
		startCh:             make(chan struct{}),
	}

	routeActivateChan <- rc

	<-rc.startCh

	rc.sgConfig.mux.HandleFunc(rc.sgConfig.Endpoint, rc.routeActiveHandler)

	ese.Log.Info().Str("event-source-name", *eventSource.Name).Str("port", sg.Port).Str("endpoint", sg.Endpoint).Msg("route handler added")

	for {
		select {
		case data := <-rc.dataCh:
			ese.Log.Info().Msg("received data")
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
			ese.Log.Info().Str("event-source-name", *eventSource.Name).Msg("connection is closed by client")
			routeDeactivateChan <- rc
			return nil

		// this error indicates that the server has stopped running
		case err := <-activeServers[rc.sgConfig.Port].errChan:
			return err
		}
	}
}

// routeActiveHandler handles new route
func (rc *routeConfig) routeActiveHandler(writer http.ResponseWriter, request *http.Request) {
	rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.sgConfig.Endpoint).Str("http-method", request.Method).Msg("received a request")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rc.eventSourceExecutor.Log.Error().Err(err).Msg("failed to parse request body")
		rc.errCh <- err
		return
	}

	rc.eventSourceExecutor.Log.Info().Str("event-source-name", *rc.eventSource.Name).Str("method", http.MethodHead).Msg("received a request")

	switch request.Method {
	case http.MethodHead:
		respBody = ""
	}
	writer.WriteHeader(http.StatusOK)
	writer.Header().Add("Content-Type", "text/plain")
	writer.Write([]byte(respBody))

	// notification received from storage grid is url encoded.
	parsedURL, err := url.QueryUnescape(string(body))
	if err != nil {
		rc.errCh <- err
		return
	}
	b, err := qson.ToJSON(parsedURL)
	if err != nil {
		rc.errCh <- err
		return
	}

	var notification *storageGridNotification
	err = json.Unmarshal(b, &notification)
	if err != nil {
		rc.errCh <- err
		return
	}

	if filterEvent(notification, rc.sgConfig) && filterName(notification, rc.sgConfig) {
		rc.eventSourceExecutor.Log.Info().Str("event-source-name", *rc.eventSource.Name).Msg("new event received, dispatching to gateway client")
		rc.dataCh <- b
		return
	}

	rc.eventSourceExecutor.Log.Warn().Str("event-source-name", *rc.eventSource.Name).Interface("notification", notification).
		Msg("discarding notification since it did not pass all filters")
}

// routeDeactivateHandler handles routes that are not active
func (rc *routeConfig) routeDeactivateHandler(writer http.ResponseWriter, request *http.Request) {
	rc.eventSourceExecutor.Log.Info().Str("endpoint", rc.sgConfig.Endpoint).Str("http-method", request.Method).Msg("route is not active")
	common.SendErrorResponse(writer)
}
