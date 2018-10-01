package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"io/ioutil"
	"net/http"
)

var (
	// gateway http server configurations
	httpGatewayServerConfig = gateways.NewHTTPGatewayServerConfig()
)

// httpConfigExecutor implements ConfigExecutor
type httpConfigExecutor struct{}

func (hce *httpConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	httpGatewayServerConfig.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("start configuration")
	err := sendHTTPRequest(&gateways.ConfigData{
		Src:    config.Data.Src,
		Config: config.Data.Config,
	}, httpGatewayServerConfig.ConfigActivateEndpoint)
	if err != nil {
		errMsg := "failed to send new configuration to gateway processor server"
		httpGatewayServerConfig.GwConfig.GatewayCleanup(config, &errMsg, err)
	}
	return err
}

func (hce *httpConfigExecutor) StopConfig(config *gateways.ConfigContext) error {
	httpGatewayServerConfig.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping configuration")
	err := sendHTTPRequest(&gateways.ConfigData{
		Src:    config.Data.Src,
		Config: config.Data.Config,
	}, httpGatewayServerConfig.ConfigurationDeactivateEndpoint)
	if err != nil {
		errMsg := "failed to send configuration to stop to gateway processor server"
		httpGatewayServerConfig.GwConfig.GatewayCleanup(config, &errMsg, err)
	}
	return err
}

func sendHTTPRequest(config *gateways.ConfigData, endpoint string) error {
	payload, err := json.Marshal(config)
	if err != nil {
		httpGatewayServerConfig.GwConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to marshal configuration")
		return err
	}
	_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", httpGatewayServerConfig.HTTPServerPort, endpoint), "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		httpGatewayServerConfig.GwConfig.Log.Warn().Str("config-key", config.Src).Err(err).Msg("failed to dispatch event to gateway-transformer.")
		return err
	}
	return nil
}

func main() {
	httpGatewayServerConfig.GwConfig.WatchGatewayEvents(context.Background())
	httpGatewayServerConfig.GwConfig.WatchGatewayConfigMap(context.Background(), &httpConfigExecutor{})
	// handle events from gateway processor server
	http.HandleFunc(httpGatewayServerConfig.EventEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		httpGatewayServerConfig.GwConfig.Log.Info().Msg("received an event. processing...")
		var event gateways.GatewayEvent
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			httpGatewayServerConfig.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		err = json.Unmarshal(body, &event)
		if err != nil {
			httpGatewayServerConfig.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		common.SendSuccessResponse(writer)
		httpGatewayServerConfig.GwConfig.DispatchEvent(&event)
	})

	httpGatewayServerConfig.GwConfig.Log.Info().Str("port", httpGatewayServerConfig.HTTPClientPort).Msg("gateway processor client is now listening for events.")
	httpGatewayServerConfig.GwConfig.Log.Fatal().Str("port", httpGatewayServerConfig.HTTPClientPort).Err(http.ListenAndServe(":"+fmt.Sprintf("%s", httpGatewayServerConfig.HTTPClientPort), nil)).Msg("")
}
