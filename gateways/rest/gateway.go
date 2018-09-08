package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"net/http"
	"os"
	"encoding/json"
	"io/ioutil"
)

var (
	// gatewayConfig provides a generic configuration for a gateway
	gatewayConfig  = gateways.NewGatewayConfiguration()

	httpServerPort = func() string {
		httpServerPort, ok := os.LookupEnv(common.GatewayProcessorServerHTTPPortEnvVar)
		if !ok {
			panic("gateway server http port is not provided")
		}
		return httpServerPort
	}()

	httpClientPort = func() string {
		httpClientPort, ok := os.LookupEnv(common.GatewayProcessorClientHTTPPortEnvVar)
		if !ok {
			panic("gateway client http port is not provided")
		}
		return httpClientPort
	}()

	eventEndpoint = func() string {
		eventEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerEventEndpointEnvVar)
		if !ok {
			panic("gateway event post endpoint is not provided")
		}
		return eventEndpoint
	}()

	configStartEndpoint = func() string {
		configStartEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStartEndpointEnvVar)
		if !ok {
			panic("gateway config start endpoint is not provided")
		}
		return configStartEndpoint
	}()

	configStopEndpoint = func() string {
		configStopEndpoint, ok := os.LookupEnv(common.GatewayProcessorHTTPServerConfigStopEndpointEnvVar)
		if !ok {
			panic("gateway config stop endpoint is not provided")
		}
		return configStopEndpoint
	}()
)

func configRunner(config *gateways.ConfigData) error {
	return sendHTTPRequest(&gateways.HTTPGatewayConfig{
		Src: config.Src,
		Config: config.Config,
	}, configStartEndpoint)
}

func configStopper(config *gateways.ConfigData) error {
	return sendHTTPRequest(&gateways.HTTPGatewayConfig{
		Src: config.Src,
		Config: config.Config,
	}, configStopEndpoint)
}

func sendHTTPRequest(config *gateways.HTTPGatewayConfig, endpoint string) error {
	payload, err := json.Marshal(config)
	if err != nil {
		gatewayConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to marshal configuration")
		return err
	}
	_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", httpServerPort, endpoint), "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		gatewayConfig.Log.Warn().Str("config-key", config.Src).Err(err).Msg("failed to dispatch event to gateway-transformer.")
		return err
	}
	return nil
}

func main() {
	gatewayConfig.WatchGatewayConfigMap(context.Background(), configRunner, configStopper)
	http.HandleFunc(eventEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		var event gateways.GatewayEvent
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			gatewayConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		err = json.Unmarshal(body, &event)
		if err != nil {
			gatewayConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		gatewayConfig.DispatchEvent(&event)
	})
	gatewayConfig.Log.Fatal().Str("port", httpClientPort).Err(http.ListenAndServe(":"+fmt.Sprintf("%s",  httpClientPort), nil)).Msg("gateway server started listening")
}
