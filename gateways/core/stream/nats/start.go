package nats

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/nats-io/go-nats"
)

// Runs a configuration
func (ce *NatsConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	defer func() {
		gateways.Recover()
	}()

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	n, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *n).Msg("nats configuration")

	go ce.listenEvents(n, config)

	for {
		select {
		case _, ok := <-config.StartChan:
			if ok {
				config.Active = true
				ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running")
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
			config.DoneChan <- struct{}{}
			ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration stopped")
			return
		}

	}
}

func (ce *NatsConfigExecutor) listenEvents(n *natsConfig, config *gateways.ConfigContext) {
	nc, err := nats.Connect(n.URL)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("url", n.URL).Err(err).Msg("connection failed")
		config.ErrChan <- err
		return
	}

	event := ce.GatewayConfig.GetK8Event("configuration running", v1alpha1.NodePhaseRunning, config.Data)
	_, err = common.CreateK8Event(event, ce.GatewayConfig.Clientset)
	if err != nil {
		ce.GatewayConfig.Log.Error().Str("config-key", config.Data.Src).Err(err).Msg("failed to mark configuration as running")
		config.ErrChan <- err
		return
	}

	config.StartChan <- struct{}{}

	_, err = nc.Subscribe(n.Subject, func(msg *nats.Msg) {
		ce.Log.Info().Str("config-key", config.Data.Src).Interface("msg", msg).Msg("received data")
		config.DataChan <- msg.Data
	})
	if err != nil {
		config.ErrChan <- err
	}
	nc.Flush()
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config", n).Msg("connected to cluster")
	if err := nc.LastError(); err != nil {
		config.ErrChan <- err
		return
	}

	<-config.DoneChan
	nc.Close()
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("configuration shutdown")
	config.ShutdownChan <- struct{}{}
}
