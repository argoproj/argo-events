package gateways

import (
	"fmt"
	"testing"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	gwFake "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	zlog "github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"time"
)

var testGateway = `
apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
  name: test-gateway
  namespace: test-namespace
  labels:
    gateways.argoproj.io/gateway-controller-instanceid: argo-events
    gateway-name: "test-gateway"
spec:
  deploySpec:
    containers:
    - name: "test-events"
      image: "argoproj/test-gateway"
      imagePullPolicy: "Always"
      command: ["/bin/test-gateway"]
    serviceAccountName: "argo-events-sa"
  configMap: "test-gateway-configmap"
  type: "test"
  dispatchMechanism: "HTTP"
  version: "1.0"
  watchers:
      gateways:
      - name: "test-gateway-2"
        port: "9070"
        endpoint: "/notifications"
      sensors:
      - name: "test-sensor"
      - name: "test-multi-sensor"
`

var testGatewayConfig = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-gateway-configmap
data:
  test.fooConfig: |-
    msg: hello`

type testConfig struct {
	Msg string
}

type testConfigExecutor struct{}

func parseConfig(config string) (*testConfig, error) {
	var t *testConfig
	err := yaml.Unmarshal([]byte(config), &t)
	if err != nil {
		return nil, err
	}
	return t, err
}

func (ce *testConfigExecutor) StartConfig(config *ConfigContext) {
	fmt.Println("operating on configuration")
	t, err := parseConfig(config.Data.Config)
	if err != nil {
		config.ErrChan <- err
		return
	}
	fmt.Println(*t)

	go ce.listenEvents(t, config)

	for {
		select {
		case <-config.StartChan:
			config.Active = true
			fmt.Println("configuration is running")

		case data := <-config.DataChan:
			fmt.Println(data)

		case <-config.StopChan:
			fmt.Println("stopping configuration")
			config.DoneChan <- struct{}{}
			fmt.Println("configuration stopped")
			config.ShutdownChan <- struct{}{}
			return
		}
	}
}

func (ce *testConfigExecutor) listenEvents(t *testConfig, config *ConfigContext) {
	config.StartChan <- struct{}{}
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Println("dispatching data")
			config.DataChan <- []byte("data")
		case <-config.DoneChan:
			fmt.Println("done with listening events.")
			return
		}
	}
}

func (ce *testConfigExecutor) Validate(config *ConfigContext) error {
	t, err := parseConfig(config.Data.Config)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrEmptyConfig
	}
	if t.Msg == "" {
		return fmt.Errorf("msg cant be empty")
	}
	return nil
}

func (ce *testConfigExecutor) StopConfig(config *ConfigContext) {
	fmt.Println("stop configuration received")
	if config.Active {
		config.Active = false
		config.StopChan <- struct{}{}
	}
}

func getGateway() (*v1alpha1.Gateway, error) {
	var gw *v1alpha1.Gateway
	err := yaml.Unmarshal([]byte(testGateway), &gw)
	if err != nil {
		return nil, err
	}
	return gw, nil
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


func Test_createInternalConfigs(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	configmap, err := gatewayConfigMap()
	assert.Nil(t, err)
	assert.NotNil(t, configmap)

	// test createInternalConfigs
	configs, err := gc.createInternalConfigs(configmap)
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

func Test_validateConfigs(t *testing.T)  {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	configmap, err := gatewayConfigMap()
	assert.Nil(t, err)
	configs, err := gc.createInternalConfigs(configmap)
	assert.Nil(t, err)
	err = gc.validateConfigs(&testConfigExecutor{}, configs)
	assert.Nil(t, err)
}

func Test_startConfigs(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	configmap, err := gatewayConfigMap()
	assert.Nil(t, err)
	configs, err := gc.createInternalConfigs(configmap)
	assert.Nil(t, err)

	_, newConfigKeys := gc.diffConfigurations(configs)

	err = gc.startConfigs(&testConfigExecutor{}, configs, newConfigKeys)
	assert.Nil(t, err)

	el, err := gc.Clientset.CoreV1().Events(gw.Namespace).List(metav1.ListOptions{
	})
	assert.Nil(t, err)
	assert.NotNil(t, el.Items)
	assert.Equal(t, string(v1alpha1.NodePhaseInitialized), el.Items[0].Action)

	// giving time to mark config as active
	time.Sleep(5 * time.Second)

	for _, value := range configs {
		fmt.Println("sending error")
		value.ErrChan <- fmt.Errorf("error")
		_, ok := <- value.DataChan
		assert.Equal(t, false, ok)
		_, ok = <-value.StopChan
		assert.Equal(t, false, ok)
		_, ok = <-value.DoneChan
		assert.Equal(t, false, ok)
		_, ok = <-value.StartChan
		assert.Equal(t, false, ok)
		_, ok = <-value.ErrChan
		assert.Equal(t, false, ok)
		_, ok = <-value.ShutdownChan
		assert.Equal(t, false, ok)
	}
}

func Test_stopConfigs(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	configmap, err := gatewayConfigMap()
	assert.Nil(t, err)
	configs, err := gc.createInternalConfigs(configmap)
	assert.Nil(t, err)

	_, newConfigKeys := gc.diffConfigurations(configs)

	err = gc.startConfigs(&testConfigExecutor{}, configs, newConfigKeys)
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)

	keys, _ := gc.diffConfigurations(make(map[string]*ConfigContext))

	fmt.Println(keys)

	gc.stopConfigs(&testConfigExecutor{}, keys)

	time.Sleep(5 * time.Second)

	for _, config := range configs {
		assert.Equal(t, false, config.Active)
	}
}
