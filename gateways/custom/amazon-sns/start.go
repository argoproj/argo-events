package amazon_sns

import "github.com/argoproj/argo-events/gateways"

func (ce *AWSSNSConfigExecutor) StartConfig(config *gateways.ConfigContext) {
	ce.GatewayConfig.Log.Info().Str("config-name", config.Data.Src).Msg("operating on configuration")
	g, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- gateways.ErrConfigParseFailed
	}
	ce.GatewayConfig.Log.Debug().Str("config-key", config.Data.Src).Interface("config-value", *g).Msg("aws sns configuration")



}

