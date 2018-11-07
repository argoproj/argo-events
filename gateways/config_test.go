package gateways

import (
	"time"
	"testing"
	"github.com/stretchr/testify/assert"
)

func Test_createInternalConfigs(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gatewayConfig := newGatewayconfig(gw)
	configmap, err := gatewayConfigMap()
	assert.Nil(t, err)
	assert.NotNil(t, configmap)

	// test createInternalConfigs
	configs, err := gatewayConfig.createInternalConfigs(configmap)
	assert.Nil(t, err)
	assert.NotNil(t, configs)

	for _, config := range configs {
		assert.NotNil(t, config.Data)
		assert.NotNil(t, config.Data.Src)
		assert.NotNil(t, config.Data.TimeID)
		assert.NotNil(t, config.Data.ID)
		assert.Equal(t, configmap.Data[config.Data.Src], config.Data.Config)
	}
}

func Test_diffConfigurations(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gatewayConfig := newGatewayconfig(gw)
	configmap, err := gatewayConfigMap()
	assert.Nil(t, err)
	assert.NotNil(t, configmap)

	// test createInternalConfigs
	configs, err := gatewayConfig.createInternalConfigs(configmap)
	assert.Nil(t, err)

	staleConfigKeys, newConfigKeys := gatewayConfig.diffConfigurations(configs)
	assert.Empty(t, staleConfigKeys)
	assert.NotNil(t, newConfigKeys)

	gatewayConfig.registeredConfigs = configs
	staleConfigKeys, newConfigKeys = gatewayConfig.diffConfigurations(configs)
	assert.Equal(t, staleConfigKeys, newConfigKeys)

	configName := "new-test-config"
	newConfigContext := &ConfigContext{
		Data: &ConfigData{
			ID:     Hasher(configName),
			TimeID: Hasher(time.Now().String()),
			Src:    "test.newConfig",
			Config: `|-
    msg: new message`,
		},
		Active: false,
		StopChan: make(chan struct{}),
		DoneChan: make(chan struct{}),
		ErrChan: make(chan error),
		DataChan: make(chan []byte),
		StartChan: make(chan struct{}),
	}

	newConfigs := map[string]*ConfigContext{
		Hasher(newConfigContext.Data.Src + newConfigContext.Data.Config): newConfigContext,
	}
	staleConfigKeys, newConfigKeys = gatewayConfig.diffConfigurations(newConfigs)
	assert.NotNil(t, staleConfigKeys)
	assert.NotEqual(t, staleConfigKeys, newConfigKeys)
}

func Test_manageConfigurations(t *testing.T) {

}
