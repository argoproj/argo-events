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
	"github.com/argoproj/argo-events/common"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/argoproj/argo-events/gateways"
	"github.com/joncalhoun/qson"
	"github.com/satori/go.uuid"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers.
	activeServers = make(map[string]*activeServer)

	// activeEndpoints keep track of endpoints that are already registered with server and their status active or deactive
	activeEndpoints = make(map[string]*endpoint)

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
	sgConfig            *storageGrid
	eventSource         *gateways.EventSource
	eventSourceExecutor *StorageGridEventSourceExecutor
	startCh             chan struct{}
}

type endpoint struct {
	active bool
	dataCh chan []byte
	errCh  chan error
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
				_, ok := activeServers[config.sgConfig.Port]
				if ok {
					activeEndpoints[config.sgConfig.Endpoint].active = false
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
func filterEvent(notification *storageGridNotification, sg *storageGrid) bool {
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
func filterName(notification *storageGridNotification, sg *storageGrid) bool {
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
			rc.eventSourceExecutor.Log.Info().Str("event-source", rc.eventSource.Name).Str("port", rc.sgConfig.Port).Msg("http server stopped")
			if err != nil {
				errChan <- err
			}
		}()
	}
	mutex.Unlock()
}

// StartConfig runs a configuration
func (ese *StorageGridEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	sg, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	rc := routeConfig{
		sgConfig:            sg,
		eventSource:         eventSource,
		eventSourceExecutor: ese,
		startCh:             make(chan struct{}),
	}

	routeActivateChan <- rc

	<-rc.startCh

	if rc.sgConfig.mux == nil {
		mutex.Lock()
		rc.sgConfig.mux = activeServers[rc.sgConfig.Port].srv
		mutex.Unlock()
	}

	ese.Log.Info().Str("event-source-name", eventSource.Name).Str("port", sg.Port).Str("endpoint", sg.Endpoint).Msg("adding route handler")
	if _, ok := activeEndpoints[rc.sgConfig.Endpoint]; !ok {
		activeEndpoints[rc.sgConfig.Endpoint] = &endpoint{
			active: true,
			dataCh: make(chan []byte),
			errCh:  make(chan error),
		}
		rc.sgConfig.mux.HandleFunc(rc.sgConfig.Endpoint, rc.routeActiveHandler)
	}
	activeEndpoints[rc.sgConfig.Endpoint].active = true

	ese.Log.Info().Str("event-source-name", eventSource.Name).Str("port", sg.Port).Str("endpoint", sg.Endpoint).Msg("route handler added")

	for {
		select {
		case data := <-activeEndpoints[rc.sgConfig.Endpoint].dataCh:
			ese.Log.Info().Msg("received data")
			err := eventStream.Send(&gateways.Event{
				Name:    eventSource.Name,
				Payload: data,
			})
			if err != nil {
				ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Str("port", sg.Port).Str("endpoint", sg.Endpoint).Msg("failed to send event")
				return err
			}

		case err := <-activeEndpoints[rc.sgConfig.Endpoint].errCh:
			ese.Log.Error().Err(err).Str("event-source-name", eventSource.Name).Str("port", sg.Port).Str("endpoint", sg.Endpoint).Msg("internal error occurred")

		case <-eventStream.Context().Done():
			ese.Log.Info().Str("event-source-name", eventSource.Name).Str("port", sg.Port).Str("endpoint", sg.Endpoint).Msg("connection is closed by client")
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
	if !activeEndpoints[rc.sgConfig.Endpoint].active {
		rc.eventSourceExecutor.Log.Info().Str("event-source-name", rc.eventSource.Name).Str("method", http.MethodHead).Msg("deactived route")
		common.SendErrorResponse(writer, "route is not valid")
		return
	}

	rc.eventSourceExecutor.Log.Info().Str("event-source-name", rc.eventSource.Name).Str("method", http.MethodHead).Msg("received a request")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rc.eventSourceExecutor.Log.Error().Err(err).Msg("failed to parse request body")
		common.SendErrorResponse(writer, "failed to parse request body")
		return
	}

	switch request.Method {
	case http.MethodHead:
		respBody = ""
	}
	writer.WriteHeader(http.StatusOK)
	writer.Header().Add("Content-Type", "text/plain")
	writer.Write([]byte(respBody))

	rc.eventSourceExecutor.Log.Info().Str("body", string(body)).Msg("response body")

	// notification received from storage grid is url encoded.
	parsedURL, err := url.QueryUnescape(string(body))
	if err != nil {
		activeEndpoints[rc.sgConfig.Endpoint].errCh <- err
		return
	}
	b, err := qson.ToJSON(parsedURL)
	if err != nil {
		activeEndpoints[rc.sgConfig.Endpoint].errCh <- err
		return
	}

	var notification *storageGridNotification
	err = json.Unmarshal(b, &notification)
	if err != nil {
		activeEndpoints[rc.sgConfig.Endpoint].errCh <- err
		return
	}

	if filterEvent(notification, rc.sgConfig) && filterName(notification, rc.sgConfig) {
		rc.eventSourceExecutor.Log.Info().Str("event-source-name", rc.eventSource.Name).Msg("new event received, dispatching to gateway client")
		activeEndpoints[rc.sgConfig.Endpoint].dataCh <- b
		return
	}

	rc.eventSourceExecutor.Log.Warn().Str("event-source-name", rc.eventSource.Name).Interface("notification", notification).
		Msg("discarding notification since it did not pass all filters")
}
