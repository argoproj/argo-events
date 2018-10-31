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
	"go.uber.org/atomic"
	"io/ioutil"
	"net/http"
	"sync"
	"strings"
	"strconv"
)

var (
	// whether http server has started or not
	hasServerStarted atomic.Bool

	// srv holds reference to http server
	srv http.Server

	// mutex synchronizes activeRoutes
	mutex sync.Mutex
	// as http package does not provide method for unregistering routes,
	// this keeps track of configured http routes and their methods.
	// keeps endpoints as keys and corresponding http methods as a map
	activeRoutes = make(map[string]map[string]struct{})

	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig = gateways.NewGatewayConfiguration()
)

// webhookConfigExecutor implements ConfigExecutor
type webhookConfigExecutor struct{}

// parseConfig parses webhook configuration
func (wce *webhookConfigExecutor) parseConfig(config *gateways.ConfigContext) (*webhook, error) {
	var h *webhook
	err := yaml.Unmarshal([]byte(config.Data.Config), &h)
	if err != nil {
		return nil, err
	}
	return nil, err
}

// Runs a gateway configuration
func (wce *webhookConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer gatewayConfig.GatewayCleanup(config, &errMessage, err)

	gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("parsing configuration...")

	h, err := wce.parseConfig(config)
	if err != nil {
		errMessage = "failed to parse configuration"
		return err
	}
	gatewayConfig.Log.Info().Interface("config", config.Data.Config).Interface("webhook", h).Msg("configuring...")

	var wg sync.WaitGroup
	wg.Add(1)

	// waits till disconnection from client and perform cleanup.
	go func() {
		<-config.StopCh
		config.Active = false
		gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping the configuration...")
		// remove the endpoint and http method configuration.
		mutex.Lock()
		activeHTTPMethods, ok := activeRoutes[h.Endpoint]
		if ok {
			delete(activeHTTPMethods, h.Method)
		}
		if h.Port != "" && hasServerStarted.Load() {
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping http server")
			err = srv.Shutdown(context.Background())
			if err != nil {
				errMessage = "failed to stop http server"
			}
		}
		mutex.Unlock()
		wg.Done()
	}()

	config.Active = true

	event := gatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, gatewayConfig.Clientset)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		return err
	}

	// start a http server only if given configuration contains port information and no other
	// configuration previously started the server
	if h.Port != "" && !hasServerStarted.Load() {
		// mark http server as started
		hasServerStarted.Store(true)
		go func() {
			gatewayConfig.Log.Info().Str("http-port", h.Port).Msg("http server started listening...")
			srv := &http.Server{Addr: ":" + fmt.Sprintf("%s", h.Port)}
			err = srv.ListenAndServe()
			gatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("http server stopped")
			if err == http.ErrServerClosed {
				err = nil
			}
			if err != nil {
				errMessage = "http server stopped"
			}
			if config.Active == true {
				config.StopCh <- struct{}{}
			}
			return
		}()
	}

	// configure endpoint and http method
	if h.Endpoint != "" && h.Method != "" {
		if _, ok := activeRoutes[h.Endpoint]; !ok {
			mutex.Lock()
			activeRoutes[h.Endpoint] = make(map[string]struct{})
			// save event channel for this connection/configuration
			activeRoutes[h.Endpoint][h.Method] = struct{}{}
			mutex.Unlock()

			// add a handler for endpoint if not already added.
			http.HandleFunc(h.Endpoint, func(writer http.ResponseWriter, request *http.Request) {
				// check if http methods match and route and http method is registered.
				if _, ok := activeRoutes[h.Endpoint]; ok {
					if _, isActive := activeRoutes[h.Endpoint][request.Method]; isActive {
						gatewayConfig.Log.Info().Str("endpoint", h.Endpoint).Str("http-method", h.Method).Msg("received a request")
						body, err := ioutil.ReadAll(request.Body)
						if err != nil {
							gatewayConfig.Log.Error().Err(err).Msg("failed to parse request body")
							common.SendErrorResponse(writer)
						} else {
							gatewayConfig.Log.Info().Str("endpoint", h.Endpoint).Str("http-method", h.Method).Msg("dispatching event to gateway-processor")
							common.SendSuccessResponse(writer)

							gatewayConfig.Log.Info().Str("payload", string(body)).Msg("payload is")

							// dispatch event to gateway transformer
							gatewayConfig.DispatchEvent(&gateways.GatewayEvent{
								Src:     config.Data.Src,
								Payload: body,
							})
						}
					} else {
						gatewayConfig.Log.Warn().Str("endpoint", h.Endpoint).Str("http-method", request.Method).Msg("endpoint and http method is not an active route")
						common.SendErrorResponse(writer)
					}
				} else {
					gatewayConfig.Log.Warn().Str("endpoint", h.Endpoint).Msg("endpoint is not active")
					common.SendErrorResponse(writer)
				}
			})
		} else {
			mutex.Lock()
			activeRoutes[h.Endpoint][h.Method] = struct{}{}
			mutex.Unlock()
		}

		gatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running...")
	}
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

// Validate validates given webhook configuration
func (wce *webhookConfigExecutor) Validate(config *gateways.ConfigContext) error {
	h, err := wce.parseConfig(config)
	if err != nil {
		return err
	}
	switch h.Method {
	case http.MethodHead, http.MethodPut, http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodTrace:
	default:
		return fmt.Errorf("unknown HTTP method %s", h.Method)
	}
	if h.Endpoint == "" {
		return fmt.Errorf("endpoint can't be empty")
	}
	if !strings.HasPrefix("/", h.Endpoint) {
		return fmt.Errorf("endpoint must start with '/'")
	}
	if h.Port != "" {
		_, err := strconv.Atoi(h.Port)
		if err != nil {
			return fmt.Errorf("failed to parse server port. err: %+v", err)
		}
	}
	return nil
}

func main() {
	gatewayConfig.StartGateway(&webhookConfigExecutor{})
}
