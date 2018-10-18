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

package main

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"net/http"
	"sync"
	"io/ioutil"
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
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// webhookConfigExecutor implements ConfigExecutor
type webhookConfigExecutor struct{}

// webhook is a general purpose REST API
type webhook struct {
	// REST API endpoint
	Endpoint string

	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string

	// Port on which HTTP server is listening for incoming events.
	Port string

	// srv holds reference to http server
	srv *http.Server
	mux *http.ServeMux
}

type server struct {
	mux *http.ServeMux
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// starts a http server
func (wce *webhookConfigExecutor) startHttpServer(hook *webhook, config *gateways.ConfigContext, err error, errMessage *string) {
	// start a http server only if no other configuration previously started the server on given port
	mutex.Lock()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("active servers", activeServers[hook.Port]).Msg("active servers")
	if _, ok := activeServers[hook.Port]; !ok {
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Str("port", hook.Port).Msg("http server started listening...")
		s := &server{
			mux: http.NewServeMux(),
		}
		hook.mux = s.mux
		hook.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", hook.Port),
			Handler: s,
		}
		activeServers[hook.Port] = s.mux

		// start http server
		go func() {
			err := hook.srv.ListenAndServe()
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("http server stopped")
			if err == http.ErrServerClosed {
				err = nil
			}
			if err != nil {
				msg := fmt.Sprintf("failed to stop http server. configuration err message: %s", errMessage)
				errMessage = &msg
			}
			if config.Active == true {
				config.StopCh <- struct{}{}
			}
			return
		}()
	}
	mutex.Unlock()
}

// Runs a gateway configuration
func (wce *webhookConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")

	var hook *webhook
	err = yaml.Unmarshal([]byte(config.Data.Config), &hook)
	if err != nil {
		errMessage = "failed to parse configuration"
		return err
	}
	gatewayConfig.Log.Info().Interface("config", config.Data.Config).Interface("webhook", hook).Msg("configuring...")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")

		// remove the endpoint.
		routesMutex.Lock()
		if _, ok := activeRoutes[hook.Port]; ok {
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("routes", activeRoutes[hook.Port]).Msg("active routes")
			delete(activeRoutes[hook.Port], hook.Endpoint)
			// Check if the endpoint in this configuration was the last of the active endpoints for the http server.
			// If so, shutdown the server.
			if len(activeRoutes[hook.Port]) == 0 {
				gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("all endpoint are deactivated, stopping http server")
				err = hook.srv.Shutdown(context.Background())
				if err != nil {
					// previous err message is useful when there was an error in configuration
					errMessage = fmt.Sprintf("failed to stop http server. configuration err message: %s", errMessage)
				}
			}
		}
		routesMutex.Unlock()
		wg.Done()
	}()
	
	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	// start a http server only if no other configuration previously started the server on given port
	wce.startHttpServer(hook, config, err, &errMessage)

	// add endpoint
	routesMutex.Lock()
	if _, ok := activeRoutes[hook.Port]; !ok {
		activeRoutes[hook.Port] = make(map[string]struct{})
	}
	if _, ok := activeRoutes[hook.Port][hook.Endpoint]; !ok {
		activeRoutes[hook.Port][hook.Endpoint] = struct{}{}

		// server with same port is already started by another configuration
		if hook.mux == nil {
			mutex.Lock()
			hook.mux = activeServers[hook.Port]
			mutex.Unlock()
		}

		// if the configuration that started the server was removed even before we had chance to regiser this endpoint against the port,
		// and it was last endpoint for port, server is now start new http server
		if hook.mux == nil {
			wce.startHttpServer(hook, config, err, &errMessage)
		}

		hook.mux.HandleFunc(hook.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
			gatewayConfig.Log.Info().Str("endpoint", hook.Endpoint).Str("http-method", hook.Method).Msg("received a request")
			// I don't like checking activeRoutes again but http won't let us delete route.
			if _, ok := activeRoutes[hook.Port][hook.Endpoint]; ok {
				if  hook.Method == request.Method {
					body, err := ioutil.ReadAll(request.Body)
					if err != nil {
						gatewayConfig.Log.Error().Err(err).Msg("failed to parse request body")
						common.SendErrorResponse(writer)
					} else {
						gatewayConfig.Log.Info().Str("endpoint", hook.Endpoint).Str("http-method", hook.Method).Msg("dispatching event to gateway-processor")
						common.SendSuccessResponse(writer)
						gatewayConfig.Log.Debug().Str("payload", string(body)).Msg("payload")
						// dispatch event to gateway transformer
						gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
							Src:     config.Data.Src,
							Payload: body,
						})
					}
				} else {
					gatewayConfig.Log.Info().Str("endpoint", hook.Endpoint).Str("expected", hook.Method).Str("actual", request.Method).Msg("http method mismatched")
					common.SendErrorResponse(writer)
				}
			}
		})
	}
	routesMutex.Unlock()
	
	wg.Wait()
	gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

// StopConfig stops a configuration
func (wce *webhookConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	if config.Active == true {
		config.StopCh <- struct{}{}
	}
	return nil
}

func main() {
	err := gatewayConfig.TransformerReadinessProbe()
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayTransformerConnection)
	}
	_, err = gatewayConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayEventWatch)
	}
	_, err = gatewayConfig.WatchGatewayConfigMap(context.Background(), &webhookConfigExecutor{})
	if err != nil {
		gatewayConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayConfigmapWatch)
	}
	select {}
}
