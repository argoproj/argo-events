package nats

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/argoproj/argo-events/common"
	natsio "github.com/nats-io/go-nats"
)

// Runs a configuration
func (ce *NatsConfigExecutor) StartConfig(config *gateways.ConfigContext) error {
	var err error
	var errMessage string

	// mark final gateway state
	defer ce.GatewayConfig.GatewayCleanup(config, &errMessage, err)

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("operating on configuration...")
	n, err := parseConfig(config.Data.Config)
	if err != nil {
		return err
	}
	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Interface("config-value", *n).Msg("nats configuration")

	go ce.listenEvents(n, config)

	for {
		select {
		case <-config.StartChan:
			config.Active = true
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is running")

		case data := <-config.DataChan:
			ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("dispatching event to gateway-transformer")
			ce.GatewayConfig.DispatchEvent(&gateways.GatewayEvent{
				Src:     config.Data.Src,
				Payload: data,
			})
		}

	}

	ce.GatewayConfig.Log.Info().Str("config-key", config.Data.Src).Msg("configuration is now complete.")
	return nil
}

func (ce *NatsConfigExecutor) listenEvents(n *nats, config *gateways.ConfigContext) {
	conn, err := natsio.Connect(n.URL)
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

	sub, err := conn.Subscribe(n.Subject, func(msg *natsio.Msg) {
		config.DataChan <- msg.Data
	})
	if err != nil {
		config.ErrChan <- err
		return
	}

	<-config.StopChan
	config.Active = false
	sub.Unsubscribe()
	config.DoneChan <- struct{}{}
}