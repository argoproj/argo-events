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

package gateways

import (
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers/gateway/transform"
	hs "github.com/mitchellh/hashstructure"
	zlog "github.com/rs/zerolog"
	"os"
)

// HTTPGatewayServerConfig contains information regarding http ports, endpoints
type HTTPGatewayServerConfig struct {
	// HTTPServerPort is the port on which gateway processor server is runnung
	HTTPServerPort string

	// HTTPClientPort is the port on which gateway processor client is running
	HTTPClientPort string

	// ConfigActivateEndpoint is REST endpoint listening for new configurations to run.
	ConfigActivateEndpoint string

	// ConfigurationDeactivateEndpoint is REST endpoint listening to deactivate active configuration
	ConfigurationDeactivateEndpoint string

	// EventEndpoint is REST endpoint on which gateway processor server sends events to gateway processor client
	EventEndpoint string

	// GwConfig holds generic gateway configuration
	GwConfig *GatewayConfig
}

// returns a new HTTPGatewayServerConfig
func NewHTTPGatewayServerConfig() *HTTPGatewayServerConfig {

	httpGatewayServerConfig := &HTTPGatewayServerConfig{}

	httpGatewayServerConfig.HTTPServerPort = func() string {
		httpServerPort, ok := os.LookupEnv(common.GatewayProcessorServerHTTPPortEnvVar)
		if !ok {
			panic("gateway server http port is not provided")
		}
		return httpServerPort
	}()

	httpGatewayServerConfig.HTTPClientPort = func() string {
		httpClientPort, ok := os.LookupEnv(common.GatewayProcessorClientHTTPPortEnvVar)
		if !ok {
			panic("gateway client http port is not provided")
		}
		return httpClientPort
	}()

	httpGatewayServerConfig.ConfigActivateEndpoint = func() string {
		configActivateEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStartEndpointEnvVar)
		if !ok {
			panic("gateway config activation endpoint is not provided")
		}
		return configActivateEndpoint
	}()

	httpGatewayServerConfig.ConfigurationDeactivateEndpoint = func() string {
		configDeactivateEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStopEndpointEnvVar)
		if !ok {
			panic("gateway config deactivation endpoint is not provided")
		}
		return configDeactivateEndpoint
	}()

	httpGatewayServerConfig.EventEndpoint = func() string {
		eventEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerEventEndpointEnvVar)
		if !ok {
			panic("gateway event endpoint is not provided")
		}
		return eventEndpoint
	}()

	httpGatewayServerConfig.GwConfig = NewGatewayConfiguration()

	return httpGatewayServerConfig
}

// TransformerPayload creates a new payload from input data and adds source information
func TransformerPayload(b []byte, source string) ([]byte, error) {
	tp := &transform.TransformerPayload{
		Src:     source,
		Payload: b,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

// Logger returns a JSON output logger.
func Logger(name string) zlog.Logger {
	return zlog.New(os.Stdout).With().Str("name", name).Logger()
}

func Hasher(key string, value string) (uint64, error) {
	return hs.Hash(key+value, &hs.HashOptions{})
}
