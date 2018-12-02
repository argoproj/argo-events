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

package aws_sns

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/AdRoll/goamz/sns"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
)

var (
	// mutex synchronizes activeServers
	mutex sync.Mutex
	// activeServers keeps track of currently running http servers and corresponding routes.
	activeServers = make(map[string]*http.ServeMux)

	// routeActivateChan handles assigning new route to server.
	routeActivateChan = make(chan routeConfig)

	routeDeactivateChan = make(chan routeConfig)
)

// server
type server struct {
	mux *http.ServeMux
}

type routeConfig struct {
	awsConfig      *AWSSNSConfig
	gatewayConfig  *gateways.ConfigContext
	configExecutor *AWSSNSConfigExecutor
}

func init() {
	go func() {
		for {
			select {
			case config := <-routeActivateChan:
				// start server if it has not been started on this port
				_, ok := activeServers[config.awsConfig.Port]
				if !ok {
					config.startHttpServer()
				}
				config.awsConfig.mux.HandleFunc(config.awsConfig.Endpoint, config.routeActiveHandler)

			case config := <-routeDeactivateChan:
				_, ok := activeServers[config.awsConfig.Port]
				if ok {
					config.awsConfig.mux.HandleFunc(config.awsConfig.Endpoint, config.routeDeactivateHandler)
				}
			}
		}
	}()
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// starts a http server
func (rc *routeConfig) startHttpServer() {
	// start a http server only if no other configuration previously started the server on given port
	mutex.Lock()
	if _, ok := activeServers[rc.awsConfig.Port]; !ok {
		rc.configExecutor.Log.Info().Str("config-key", rc.gatewayConfig.Data.Src).Str("port", rc.awsConfig.Port).Msg("http server will start listening")
		s := &server{
			mux: http.NewServeMux(),
		}
		rc.awsConfig.mux = s.mux
		rc.awsConfig.srv = &http.Server{
			Addr:    ":" + fmt.Sprintf("%s", rc.awsConfig.Port),
			Handler: s,
		}
		activeServers[rc.awsConfig.Port] = s.mux

		// start http server
		go func() {
			err := rc.awsConfig.srv.ListenAndServe()
			rc.configExecutor.Log.Info().Str("config-key", rc.gatewayConfig.Data.Src).Str("port", rc.awsConfig.Port).Msg("http server stopped")
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
func (ce *AWSSNSConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer func() {
		gateways.Recover()
	}()

	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("operating on configuration")
	ac, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *ac).Msg("aws sns server configuration")

	go ce.listenEvents(ac, config)

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
				awsConfig:      ac,
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
	rc.configExecutor.Log.Info().Str("endpoint", rc.awsConfig.Endpoint).Str("http-method", request.Method).Msg("received a request")
	common.SendSuccessResponse(writer)
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		rc.configExecutor.Log.Error().Err(err).Msg("failed to parse request body")
		return
	}

	var snspayload *sns.HttpNotification
	err = yaml.Unmarshal(body, &snspayload)
	if err != nil {
		rc.configExecutor.Log.Error().Err(err).Str("endpoint", rc.awsConfig.Endpoint).Str("http-method", request.Method).
			Str("payload", string(body)).Msg("failed to convert request payload into sns payload")
		return
	}

	switch snspayload.Type {
	case sns.MESSAGE_TYPE_SUBSCRIPTION_CONFIRMATION:
		client := http.Client{}
		_, err := client.Get(snspayload.SubscribeURL)
		if err != nil {
			rc.configExecutor.Log.Error().Err(err).Str("http-method", request.Method).Str("payload", string(body)).Msg("failed to send confirmation response to amazon")
			return
		}
	case sns.MESSAGE_TYPE_NOTIFICATION:
		switch rc.awsConfig.CompletePayload {
		case true:
			rc.gatewayConfig.DataChan <- body
		case false:
			rc.gatewayConfig.DataChan <- []byte(snspayload.Message)
		}
	}
}

// routeDeactivateHandler handles routes that are not active
func (rc *routeConfig) routeDeactivateHandler(writer http.ResponseWriter, request *http.Request) {
	rc.configExecutor.Log.Info().Str("endpoint", rc.awsConfig.Endpoint).Str("http-method", request.Method).Msg("route is not active")
	common.SendErrorResponse(writer)
}

func (ce *AWSSNSConfigExecutor) listenEvents(ac *AWSSNSConfig, config *gateways.ConfigContext) {
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
		awsConfig:      ac,
		gatewayConfig:  config,
		configExecutor: ce,
	}

	<-config.DoneChan
	ce.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is stopped")
	config.ShutdownChan <- struct{}{}
}
