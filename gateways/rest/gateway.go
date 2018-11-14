package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	"net/http"
	"io/ioutil"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
)

// httpConfigExecutor implements ConfigExecutor
type httpConfigExecutor struct{
	*gateways.HttpGatewayServerConfig

}

func (ce *httpConfigExecutor) StartConfig(config *gateways.ConfigContext) {

	errChan := make(chan error)

	http.HandleFunc(ce.EventEndpoint, func (writer http.ResponseWriter, request *http.Request) {
		ce.GwConfig.Log.Info().Msg("received an event")
		var event gateways.GatewayEvent
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			ce.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		err = json.Unmarshal(body, &event)
		if err != nil {
			ce.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		common.SendSuccessResponse(writer)
		ce.GwConfig.DispatchEvent(&event)
	})

	http.HandleFunc(ce.ConfigActivatedEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		ce.GwConfig.Log.Info().Msg("config running notification")
		var c *gateways.ConfigData
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			ce.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		err = json.Unmarshal(body, &c)
		if err != nil {
			ce.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			return
		}
		common.SendSuccessResponse(writer)
		event := ce.GwConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, c)
		_, err = common.CreateK8Event(event, ce.GwConfig.Clientset)
		if err != nil {
			ce.GwConfig.Log.Error().Str("config-key", c.Src).Err(err).Msg("failed to mark configuration as running")
		}
		config.StartChan <- struct{}{}
	})

	http.HandleFunc(ce.ConfigErrorEndpoint, func(writer http.ResponseWriter, request *http.Request) {
		ce.GwConfig.Log.Info().Msg("config error notification")
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			ce.GwConfig.Log.Error().Err(err).Msg("failed to read request body")
			common.SendErrorResponse(writer)
			errChan <- fmt.Errorf("error occurred in configuration")
			return
		}
		errChan <- fmt.Errorf(string(body))
		return
	})

	ce.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching configuration configuration to start")
	err := ce.sendHttpRequest(&gateways.ConfigData{
		ID: config.Data.ID,
		TimeID: config.Data.TimeID,
		Src:    config.Data.Src,
		Config: config.Data.Config,
	}, ce.StartConfigEndpoint)
	if err != nil {
		config.ErrChan <- err
	}

	for {
		select {
		case <-config.StartChan:
			config.Active = true
			ce.GwConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration is running")

		case err := <-errChan:
			config.ErrChan <- err

		case <-config.StopChan:
			ce.GwConfig.Log.Info().Str("config-name", config.Data.Src).Msg("stopping configuration")
			config.DoneChan <- struct{}{}
			ce.GwConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}
	}
}

func (ce *httpConfigExecutor) StopConfig(config *gateways.ConfigContext) {
	ce.GwConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching stop configuration notification")
	err := ce.sendHttpRequest(&gateways.ConfigData{
		Src:    config.Data.Src,
		Config: config.Data.Config,
	}, ce.StopConfigEndpoint)
	if err != nil {
		config.ErrChan <- err
	}
}


func (ce *httpConfigExecutor) Validate(configContext *gateways.ConfigContext) error {
	return nil
}

func (ce *httpConfigExecutor) sendHttpRequest(config *gateways.ConfigData, endpoint string) error {
	payload, err := json.Marshal(config)
	if err != nil {
		ce.GwConfig.Log.Error().Str("config-key", config.Src).Err(err).Msg("failed to marshal configuration")
		return err
	}
	_, err = http.Post(fmt.Sprintf("http://localhost:%s%s", ce.HttpServerPort, endpoint), "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		ce.GwConfig.Log.Warn().Str("config-key", config.Src).Err(err).Msg("failed to dispatch event to gateway")
		return err
	}
	return nil
}

func main() {
	gc := gateways.NewHttpGatewayServerConfig()
	err := gc.GwConfig.TransformerReadinessProbe()
	if err != nil {
		gc.GwConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayTransformerConnectionMsg)
	}
	_, err = gc.GwConfig.WatchGatewayEvents(context.Background())
	if err != nil {
		gc.GwConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayEventWatchMsg)
	}
	_, err = gc.GwConfig.WatchGatewayConfigMap(context.Background(), &httpConfigExecutor{
		gc,
	})
	if err != nil {
		gc.GwConfig.Log.Panic().Err(err).Msg(gateways.ErrGatewayConfigmapWatchMsg)
	}
	gc.GwConfig.Log.Info().Str("port", gc.HttpClientPort).Msg("gateway processor client is running")
	gc.GwConfig.Log.Fatal().Err(http.ListenAndServe(":"+fmt.Sprintf("%s", gc.HttpClientPort), nil)).Msg("")
}
