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
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/joncalhoun/qson"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers.
	activeServers = make(map[string]*http.ServeMux)

	// mutex synchronizes activeRoutes
	routesMutex sync.Mutex
	// activeRoutes keep track of active routes for a http server
	activeRoutes = make(map[string]map[string]struct{})

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

// starts a http server
func (ce *StorageGridConfigExecutor) startHttpServer(sg *StorageGridEventConfig, config *gateways.ConfigContext) {
	// start a http server only if no other configuration previously started the server on given port
	mutex.Lock()

	ce.Log.Info().
		Str("config-key", config.Data.Src).
		Interface("active-servers", activeServers[sg.Port]).
		Msg("servers")

	if _, ok := activeServers[sg.Port]; !ok {
		ce.Log.Info().
			Str("config-key", config.Data.Src).
			Str("port", sg.Port).
			Msg("http server started listening...")

		s := &server{
			mux: http.NewServeMux(),
		}
		sg.Mux = s.mux
		sg.Srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", sg.Port),
			Handler: s,
		}
		activeServers[sg.Port] = s.mux

		// start http server
		go func() {
			err := sg.Srv.ListenAndServe()
			ce.Log.Info().
				Str("config-key", config.Data.Src).Msg("http server stopped")

			if err == http.ErrServerClosed {
				err = nil
				return
			}
			if err != nil {
				config.ErrChan <- err
				return
			}
		}()
	}
	mutex.Unlock()
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

// StartConfig runs a configuration
func (ce *StorageGridConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("operating on configuration")
	sg, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *sg).Msg("storage grid configuration")

	for {
		select {
		case <-config.StartChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
			config.Active = true

		case data := <-config.DataChan:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-processor")
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			ce.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")

			// remove the endpoint.
			routesMutex.Lock()
			if _, ok := activeRoutes[sg.Port]; ok {
				ce.Log.Info().Str("config-key", config.Data.Src).Interface("routes", activeRoutes[sg.Port]).Msg("active routes")
				delete(activeRoutes[sg.Port], sg.Endpoint)
				// Check if the endpoint in this configuration was the last of the active endpoints for the http server.
				// If so, shutdown the server.
				if len(activeRoutes[sg.Port]) == 0 {
					ce.Log.Info().Str("config-key", config.Data.Src).Msg("all endpoint are deactivated, stopping http server")
					err = sg.Srv.Shutdown(context.Background())
					if err != nil {
						ce.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("error occurred while shutting server down")
					}
				}
			}
			routesMutex.Unlock()

			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *StorageGridConfigExecutor) listenEvents(sg *StorageGridEventConfig, config *gateways.ConfigContext) {
	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err := common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	// start a http server only if no other configuration previously started the server on given port
	ce.startHttpServer(sg, config)

	// add endpoint
	routesMutex.Lock()
	if _, ok := activeRoutes[sg.Port]; !ok {
		activeRoutes[sg.Port] = make(map[string]struct{})
	}

	if _, ok := activeRoutes[sg.Port][sg.Endpoint]; !ok {
		activeRoutes[sg.Port][sg.Endpoint] = struct{}{}

		// server with same port is already started by another configuration
		if sg.Mux == nil {
			mutex.Lock()
			sg.Mux, ok = activeServers[sg.Port]
			if !ok {
				ce.startHttpServer(sg, config)
			}
			mutex.Unlock()
		}

		sg.Mux.HandleFunc(sg.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
			ce.Log.Info().Str("endpoint", sg.Endpoint).Str("http-method", request.Method).Msg("received a request")
			// Todo: find a better to handle route deletion
			if _, ok := activeRoutes[sg.Port][sg.Endpoint]; ok {
				body, err := ioutil.ReadAll(request.Body)
				if err != nil {
					ce.Log.Error().Err(err).Msg("failed to parse request body")
					common.SendErrorResponse(writer)
					config.Active = false
					config.ErrChan <- err
					return
				}

				ce.Log.Info().Str("endpoint", sg.Endpoint).Str("http-method", request.Method).
					Str("payload", string(body)).Msg("dispatching event to gateway-processor")

				switch request.Method {
				case http.MethodPost, http.MethodPut:
					body, err := ioutil.ReadAll(request.Body)
					if err != nil {
						ce.Log.Error().Err(err).Str("config-key", config.Data.Src).Msg("failed to parse request body")
					} else {
						ce.Log.Info().Str("config-key", config.Data.Src).Str("msg", string(body)).Msg("msg body")
					}

				case http.MethodHead:
					ce.Log.Info().Str("config-key", config.Data.Src).Str("method", http.MethodHead).Msg("received a request")
					respBody = ""
				}
				writer.WriteHeader(http.StatusOK)
				writer.Header().Add("Content-Type", "text/plain")
				writer.Write([]byte(respBody))

				// notification received from storage grid is url encoded.
				parsedURL, err := url.QueryUnescape(string(body))
				if err != nil {
					config.Active = false
					config.ErrChan <- err
					return
				}
				b, err := qson.ToJSON(parsedURL)
				if err != nil {
					config.Active = false
					config.ErrChan <- err
					return
				}

				var notification *storageGridNotification
				err = json.Unmarshal(b, &notification)
				if err != nil {
					config.Active = false
					config.ErrChan <- err
					return
				}

				ce.Log.Info().Str("config-key", config.Data.Src).Interface("notification", notification).Msg("parsed notification")
				if filterEvent(notification, sg) && filterName(notification, sg) {
					config.DataChan <- b
				} else {
					ce.Log.Warn().Str("config-key", config.Data.Src).Interface("notification", notification).Msg("discarding notification since it did not pass all filters")
				}
			}
		})
	}
	routesMutex.Unlock()
}
