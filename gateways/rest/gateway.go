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
	gc = gateways.NewHTTPGatewayServerConfig()
)

// httpConfigExecutor implements ConfigExecutor
type httpConfigExecutor struct{}

func (ce *httpConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	gc.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("start configuration")
	err := sendHTTPRequest(&gateways.ConfigData{
		Src:    config.Data.Src,
		Config: config.Data.Config,
	}, gc.ConfigActivateEndpoint)
	if err != nil {
		config.ErrChan <- err
	}
}

func (ce *httpConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	gc.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("stopping configuration")
	err := sendHTTPRequest(&gateways.ConfigData{
		Src:    config.Data.Src,
		Config: config.Data.Config,
	}, gc.ConfigurationDeactivateEndpoint)
	if err != nil {
		config.ErrChan <- err
	}
}

func (ce *httpConfigExecutor) Validate(configContext *gateways.ConfigContext) error {
	return nil
}

func sendHTTPRequest(config *gateways.ConfigData, endpoint string) error {
	payload, err := json.Marshal(config)
	if err != nil {
		gc.GwConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to marshal configuration")
		return err
	}
	_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", gc.HTTPServerPort, endpoint), "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		gc.GwConfig.Log.Warn().Str("config-key", config.Src).Err(err).Msg("failed to dispatch event to gateway-transformer.")
		return err
	}
	return nil
}

func main() {
	err := gc.GwConfig.TransformerReadinessProbe()
	if err != nil {
		gc.GwConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayTransformerConnectionMsg)
	}
	_, err = gc.GwConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gc.GwConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayEventWatchMsg)
	}
	_, err = gc.GwConfig.WatchGatewayConfigMap(context.Background(), &httpConfigExecutor{})
	if err != nil {
		gc.GwConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayConfigmapWatchMsg)
	}
	// handle events from gateway processor server
	http.HandleFunc(gc.EventEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		gc.GwConfig.Log.Info().Msg("received an event. processing...")
		var event gateways.GatewayEvent
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			gc.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		err = json.Unmarshal(body, &event)
		if err != nil {
			gc.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		common.SendSuccessResponse(writer)
		gc.GwConfig.DispatchEvent(&event)
	})

	gc.GwConfig.Log.Fatal().Str("port", gc.HTTPClientPort).Err(http.ListenAndServe(":"+fmt.Sprintf("%s", gc.HTTPClientPort), nil)).Msg("gateway processor client is now listening for events")
}
