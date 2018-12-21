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
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/joncalhoun/qson"
	"github.com/satori/go.uuid"
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
	sgConfig       *StorageGridEventConfig
	gatewayConfig  *gateways.ConfigContext
	configExecutor *StorageGridConfigExecutor
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
		rc.configExecutor.Log.Info().Str("config-key", rc.gatewayConfig.Data.Src).Str("port", rc.sgConfig.Port).Msg("http server will start listening")
		s := &server{
			mux: http.NewServeMux(),
		}
		rc.sgConfig.mux = s.mux
		rc.sgConfig.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", rc.sgConfig.Port),
			Handler: s,
		}
		activeServers[rc.sgConfig.Port] = s.mux

		// start http server
		go func() {
			err := rc.sgConfig.srv.ListenAndServe()
			rc.configExecutor.Log.Info().Str("config-key", rc.gatewayConfig.Data.Src).Str("port", rc.sgConfig.Port).Msg("http server stopped")
			if err == http.ErrServerClosed {
				err = nil
				return
			}
			if err != nil {
				rc.gatewayConfig.ErrChan <- err
				return
			}
		}()
	}
	mutex.Unlock()
}

// StartConfig runs a configuration
func (ce *StorageGridConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer func() {
		gateways.Recover()
	}()

	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("operating on configuration")
	sg, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *sg).Msg("storage grid configuration")

	go ce.listenEvents(sg, config)

	for {
		select {
		case _, ok := <-config.StartChan:
			if ok {
				ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")
				config.Active = true
			}

		case data, ok := <-config.DataChan:
			if ok {
				err := ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
					Src:     config.Data.Src,
					Payload: data,
				})
				if err != nil {
					config.ErrChan <- err
				}
			}

		case <-config.StopChan:
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			ce.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")

			// remove the endpoint.
			routeDeactivateChan <- routeConfig{
				sgConfig:       sg,
				gatewayConfig:  config,
				configExecutor: ce,
			}

			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")

			return
		}
	}
}

// routeActiveHandler handles new route
func (rc *routeConfig) routeActiveHandler(writer http.ResponseWriter, request *http.Request) {
	rc.configExecutor.Log.Info().Str("endpoint", rc.sgConfig.Endpoint).Str("http-method", request.Method).Msg("received a request")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rc.configExecutor.Log.Error().Err(err).Msg("failed to parse request body")
		rc.gatewayConfig.ErrChan <- err
		return
	}

	switch request.Method {
	case http.MethodPost, http.MethodPut:
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			rc.configExecutor.Log.Error().Err(err).Str("config-key", rc.gatewayConfig.Data.Src).Msg("failed to parse request body")
		} else {
			rc.configExecutor.Log.Info().Str("config-key", rc.gatewayConfig.Data.Src).Str("msg", string(body)).Msg("msg body")
		}

	case http.MethodHead:
		rc.configExecutor.Log.Info().Str("config-key", rc.gatewayConfig.Data.Src).Str("method", http.MethodHead).Msg("received a request")
		respBody = ""
	}
	writer.WriteHeader(http.StatusOK)
	writer.Header().Add("Content-Type", "text/plain")
	writer.Write([]byte(respBody))

	// notification received from storage grid is url encoded.
	parsedURL, err := url.QueryUnescape(string(body))
	if err != nil {
		rc.gatewayConfig.ErrChan <- err
		return
	}
	b, err := qson.ToJSON(parsedURL)
	if err != nil {
		rc.gatewayConfig.ErrChan <- err
		return
	}

	var notification *storageGridNotification
	err = json.Unmarshal(b, &notification)
	if err != nil {
		rc.gatewayConfig.ErrChan <- err
		return
	}

	rc.configExecutor.Log.Info().Str("config-key", rc.gatewayConfig.Data.Src).Interface("notification", notification).Msg("parsed notification")
	if filterEvent(notification, rc.sgConfig) && filterName(notification, rc.sgConfig) {
		rc.gatewayConfig.DataChan <- b
		return
	}

	rc.configExecutor.Log.Warn().Str("config-key", rc.gatewayConfig.Data.Src).Interface("notification", notification).
		Msg("discarding notification since it did not pass all filters")
}

// routeDeactivateHandler handles routes that are not active
func (rc *routeConfig) routeDeactivateHandler(writer http.ResponseWriter, request *http.Request) {
	rc.configExecutor.Log.Info().Str("endpoint", rc.sgConfig.Endpoint).Str("http-method", request.Method).Msg("route is not active")
	common.SendErrorResponse(writer)
}

func (ce *StorageGridConfigExecutor) listenEvents(sg *StorageGridEventConfig, config *gateways.ConfigContext) {
	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err := common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	// at this point configuration is successfully running
	config.StartChan <- struct{}{}

	routeActivateChan <- routeConfig{
		sgConfig:       sg,
		gatewayConfig:  config,
		configExecutor: ce,
	}

	<-config.DoneChan
	ce.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is stopped")
	config.ShutdownChan <- struct{}{}
}
