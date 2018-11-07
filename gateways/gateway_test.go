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

//func Test_gatewayOperations(t *testing.T) {
//	gw, err := getGateway()
//	assert.Nil(t, err)
//	assert.NotNil(t, gw)
//	gatewayConfig := newGatewayconfig(gw)
//	configmap, err := gatewayConfigMap()
//	assert.Nil(t, err)
//	assert.NotNil(t, configmap)
//
//	// test createInternalConfigs
//	configs, err := gatewayConfig.createInternalConfigs(configmap)
//	assert.Nil(t, err)
//	assert.NotNil(t, configs)
//
//	for _, config := range configs {
//		assert.NotNil(t, config.Data)
//		assert.NotNil(t, config.Data.Src)
//		assert.NotNil(t, config.Data.TimeID)
//		assert.NotNil(t, config.Data.ID)
//		assert.Equal(t, configmap.Data[config.Data.Src], config.Data.Config)
//	}
//
//	staleConfigKeys, newConfigKeys := gatewayConfig.diffConfigurations(configs)
//	assert.Empty(t, staleConfigKeys)
//	assert.NotNil(t, newConfigKeys)
//
//	gatewayConfig.registeredConfigs = configs
//	staleConfigKeys, newConfigKeys = gatewayConfig.diffConfigurations(configs)
//	assert.Equal(t, staleConfigKeys, newConfigKeys)
//
//	// test diffConfigs
//	configName := "new-test-config"
//	newConfigContext := &ConfigContext{
//		Data: &ConfigData{
//			ID:     Hasher(configName),
//			TimeID: Hasher(time.Now().String()),
//			Src:    "test.newConfig",
//			Config: `|-
//    msg: new message`,
//		},
//		Active: false,
//		StopChan: make(chan struct{}),
//		DoneChan: make(chan struct{}),
//		ErrChan: make(chan error),
//		DataChan: make(chan []byte),
//		StartChan: make(chan struct{}),
//	}
//
//	newConfigs := map[string]*ConfigContext{
//		Hasher(newConfigContext.Data.Src + newConfigContext.Data.Config): newConfigContext,
//	}
//	staleConfigKeys, newConfigKeys = gatewayConfig.diffConfigurations(newConfigs)
//	assert.NotNil(t, staleConfigKeys)
//	assert.NotEqual(t, staleConfigKeys, newConfigKeys)
//
//	gatewayConfig.registeredConfigs = make(map[string]*ConfigContext)
//	err = gatewayConfig.manageConfigurations(&testConfigExecutor{}, configmap)
//	assert.Nil(t, err)
//
//	events, err := gatewayConfig.Clientset.CoreV1().Events("test-namespace").List(metav1.ListOptions{})
//	assert.Nil(t, err)
//	assert.NotNil(t, events)
//
//	delete(configmap.Data, "test.fooConfig")
//	err = gatewayConfig.manageConfigurations(&testConfigExecutor{}, configmap)
//	assert.Nil(t, err)
//
//	nodeStatus := gatewayConfig.initializeNode(Hasher("test-node"), "test-node", Hasher(time.Now().String()), "init")
//	gw.Status.Nodes[nodeStatus.ID] = nodeStatus
//	nodeStatus2 := gatewayConfig.MarkGatewayNodePhase(nodeStatus.ID, v1alpha1.NodePhaseInitialized, "init")
//	assert.Equal(t, string(nodeStatus.Phase), string(nodeStatus2.Phase))
//	nodeStatus2 = gatewayConfig.MarkGatewayNodePhase(nodeStatus.ID, v1alpha1.NodePhaseError, "init")
//	assert.NotEqual(t, string(nodeStatus.Phase), string(nodeStatus2.Phase))
//}
