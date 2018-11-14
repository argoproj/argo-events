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

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/gateways/rest/calendar"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

var (
	mut sync.Mutex
	// activeConfigs keeps track of configurations that are running in gateway.
	activeConfigs = make(map[string]*gateways.ConfigContext)
)

// returns a gateway configuration and its hash
func getConfiguration(body io.ReadCloser) (*gateways.ConfigContext, *string, error) {
	var configData gateways.ConfigData
	config, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read requested config to run. err %+v", err)
	}
	err = json.Unmarshal(config, &configData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config. err %+v", err)
	}
	// register configuration
	gatewayConfig := &gateways.ConfigContext{
		Data: &gateways.ConfigData{
			Config: configData.Config,
			Src:    configData.Src,
		},
		StopChan: make(chan struct{}),
	}
	hash := gateways.Hasher(configData.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash configuration. err %+v", err)
	}
	return gatewayConfig, &hash, nil
}

func main() {

	ce := &calendar.CalendarConfigExecutor{
		gateways.NewHTTPGatewayServerConfig(),
	}

	// handles new configuration. adds a stop channel to configuration, so we can pass stop signal
	// in the event of configuration deactivation
	http.HandleFunc(ce.ConfigActivateEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		config, hash, err := getConfiguration(request.Body)
		if err != nil {
			ce.GwConfig.Log.Error().Err(err).Msg("failed to start configuration")
			common.SendErrorResponse(writer)
			return
		}
		mut.Lock()
		activeConfigs[*hash] = config
		mut.Unlock()
		common.SendSuccessResponse(writer)
		go ce.StartGateway(config)
	})

	// handles configuration deactivation. no need to remove the configuration from activeConfigs as we are
	// always overriding configurations in configuration activation.
	http.HandleFunc(ce.ConfigurationDeactivateEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		_, hash, err := getConfiguration(request.Body)
		if err != nil {
			ce.GwConfig.Log.Error().Err(err).Msg("failed to stop configuration")
			common.SendErrorResponse(writer)
			return
		}
		mut.Lock()
		config, ok := activeConfigs[*hash]
		if !ok {
			ce.GwConfig.Log.Warn().Interface("config", *hash).Msg("unknown configuration to stop")
			common.SendErrorResponse(writer)
			return
		}
		ce.StopConfig(config)
		delete(activeConfigs, *hash)
		mut.Unlock()
		common.SendSuccessResponse(writer)
	})

	// start http server
	ce.GwConfig.Log.Fatal().Str("port", ce.HTTPServerPort).Err(http.ListenAndServe(":"+fmt.Sprintf("%s", ce.HTTPServerPort), nil)).Msg("gateway server started listening")
}
