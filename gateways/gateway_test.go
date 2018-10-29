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
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwFake "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	zlog "github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"sync"
	"testing"
	"time"
)

type testConfigExecutor struct{}

func (tce *testConfigExecutor) StartConfig(ctx *ConfigContext) error {
	var wg sync.WaitGroup
	wg.Add(1)
	ctx.Active = true
	fmt.Println("hello")
	go func() {
		<-ctx.StopCh
		ctx.Active = false
		fmt.Println("stopped")
		wg.Done()
	}()
	wg.Wait()
	return nil
}

func (tce *testConfigExecutor) StopConfig(ctx *ConfigContext) error {
	if ctx.Active {
		ctx.StopCh <- struct{}{}
	}
	return nil
}

var testGateway = `apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
  name: calendar-gateway
  labels:
    gateways.argoproj.io/gateway-controller-instanceid: argo-events
    gateway-name: "calendar-gateway"
spec:
  deploySpec:
    containers:
    - name: "calendar-events"
      image: "argoproj/calendar-gateway"
      imagePullPolicy: "Always"
      command: ["/bin/calendar-gateway"]
    serviceAccountName: "argo-events-sa"
  configMap: "calendar-gateway-configmap"
  type: "calendar"
  dispatchMechanism: "HTTP"
  version: "1.0"
  watchers:
      gateways:
      - name: "webhook-gateway-2"
        port: "9070"
        endpoint: "/notifications"
      sensors:
      - name: "calendar-sensor"
      - name: "multi-signal-sensor"
`

var testGatewayConfig = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-gateway-configmap
data:
  test.barConfig: |-
    interval: 55s
  test.fooConfig: |-
    interval: 10s`

func getGateway() (*v1alpha1.Gateway, error) {
	var gw v1alpha1.Gateway
	err := yaml.Unmarshal([]byte(testGateway), &gw)
	if err != nil {
		return nil, err
	}
	return &gw, nil
}

func gatewayConfigMap() (*corev1.ConfigMap, error) {
	var gconfig corev1.ConfigMap
	err := yaml.Unmarshal([]byte(testGatewayConfig), &gconfig)
	if err != nil {
		return nil, err
	}
	return &gconfig, err
}

func newGatewayconfig(gw *v1alpha1.Gateway) *GatewayConfig {
	return &GatewayConfig{
		Log:                  zlog.New(os.Stdout).With().Caller().Logger(),
		Name:                 "test-gateway",
		Namespace:            "test-namespace",
		Clientset:            fake.NewSimpleClientset(),
		controllerInstanceID: "test-id",
		configName:           "test-gateway-configmap",
		gwcs:                 gwFake.NewSimpleClientset(),
		registeredConfigs:    make(map[string]*ConfigContext),
		transformerPort:      "9000",
		gw:                   gw,
	}
}

func Test_gatewayOperations(t *testing.T) {
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

	staleConfigKeys, newConfigKeys := gatewayConfig.diffConfigurations(configs)
	assert.Empty(t, staleConfigKeys)
	assert.NotNil(t, newConfigKeys)

	gatewayConfig.registeredConfigs = configs
	staleConfigKeys, newConfigKeys = gatewayConfig.diffConfigurations(configs)
	assert.Equal(t, staleConfigKeys, newConfigKeys)

	// test diffConfigs
	configName := "new-test-config"
	newConfigContext := &ConfigContext{
		Data: &ConfigData{
			ID:     Hasher(configName),
			TimeID: Hasher(time.Now().String()),
			Src:    "test.newConfig",
			Config: `|-
    interval: 55s`,
		},
		Active: false,
		StopCh: make(chan struct{}),
	}

	newConfigs := map[string]*ConfigContext{
		Hasher(newConfigContext.Data.Src + newConfigContext.Data.Config): newConfigContext,
	}
	staleConfigKeys, newConfigKeys = gatewayConfig.diffConfigurations(newConfigs)
	assert.NotNil(t, staleConfigKeys)
	assert.NotEqual(t, staleConfigKeys, newConfigKeys)

	gatewayConfig.registeredConfigs = make(map[string]*ConfigContext)
	err = gatewayConfig.manageConfigurations(&testConfigExecutor{}, configmap)
	assert.Nil(t, err)

	events, err := gatewayConfig.Clientset.CoreV1().Events("test-namespace").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, events)

	delete(configmap.Data, "test.fooConfig")
	err = gatewayConfig.manageConfigurations(&testConfigExecutor{}, configmap)
	assert.Nil(t, err)

	nodeStatus := gatewayConfig.initializeNode(Hasher("test-node"), "test-node", Hasher(time.Now().String()), "init")
	gw.Status.Nodes[nodeStatus.ID] = nodeStatus
	nodeStatus2 := gatewayConfig.MarkGatewayNodePhase(nodeStatus.ID, v1alpha1.NodePhaseInitialized, "init")
	assert.Equal(t, string(nodeStatus.Phase), string(nodeStatus2.Phase))
	nodeStatus2 = gatewayConfig.MarkGatewayNodePhase(nodeStatus.ID, v1alpha1.NodePhaseError, "init")
	assert.NotEqual(t, string(nodeStatus.Phase), string(nodeStatus2.Phase))
}
