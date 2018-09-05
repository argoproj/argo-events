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

package core

import (
	"bytes"
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways/utils"
	hs "github.com/mitchellh/hashstructure"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"os"
)

// gatewayConfig provides a generic configuration for a gateway
type GatewayConfig struct {
	// log provides fast and simple logger dedicated to JSON output
	log zerolog.Logger
	// Clientset is client for kubernetes API
	Clientset *kubernetes.Clientset
	// Namespace is namespace for the gateway to run inside
	Namespace string
	// transformerPort is gateway transformer port to dispatch event to
	transformerPort string
	// registeredConfigs stores information about current configurations that are running in the gateway
	registeredConfigs map[uint64]*ConfigData
	// configName is name of configmap that contains run configuration/s for the gateway
	configName string
}

type ConfigData struct {
	Src    string
	Config string
	StopCh chan struct{}
	Active bool
}

type GatewayExecutor interface {
	RunConfiguration(config *ConfigData) error
}

// Watches change in configuration for the gateway
func (gc *GatewayConfig) WatchGatewayConfigMap(gtEx GatewayExecutor, ctx context.Context) (cache.Controller, error) {
	source := gc.newConfigMapWatch(gc.configName)
	_, controller := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*corev1.ConfigMap); ok {
					gc.log.Info().Str("config-map", gc.configName).Msg("detected ConfigMap addition. Updating the controller run config.")
					err := gc.manageConfigurations(gtEx, cm)
					if err != nil {
						gc.log.Error().Err(err).Msg("update of run config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*corev1.ConfigMap); ok {
					gc.log.Info().Msg("detected ConfigMap update. Updating the controller run config.")
					err := gc.manageConfigurations(gtEx, newCm)
					if err != nil {
						gc.log.Error().Err(err).Msg("update of run config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// creates a new configmap watch
func (gc *GatewayConfig) newConfigMapWatch(name string) *cache.ListWatch {
	x := gc.Clientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// syncs registered configurations and updated gateway configmap
func (gc *GatewayConfig) manageConfigurations(gtEx GatewayExecutor, cm *corev1.ConfigMap) error {
	newConfigs, err := gc.createInternalConfigs(cm)
	if err != nil {
		return err
	}
	staleConfigKeys, newConfigKeys := gc.diffConfigurations(newConfigs)

	gc.log.Debug().Interface("stale-config-keys", staleConfigKeys).Msg("stale config")
	gc.log.Debug().Interface("new-config-keys", newConfigKeys).Msg("new config")

	// run new configurations
	for _, newConfigKey := range newConfigKeys {
		newConfig := newConfigs[newConfigKey]
		gc.registeredConfigs[newConfigKey] = newConfig
		// run configuration
		gc.log.Info().Str("config-key", newConfig.Src).Msg("running gateway...")
		go gtEx.RunConfiguration(newConfig)
	}

	// remove stale configurations
	for _, staleConfigKey := range staleConfigKeys {
		staleConfig := gc.registeredConfigs[staleConfigKey]

		// send a stop signal to configuration.
		// it is run a separate go routine because in event of RunConfiguration throwing an exception, we have
		// already closed the configuration, so there is no listening on other end of channel.
		if staleConfig.Active {
			staleConfig.StopCh <- struct{}{}
		}

		gc.log.Info().Str("config", staleConfig.Src).Msg("deleting configuration...")
		delete(gc.registeredConfigs, staleConfigKey)
	}
	return nil
}

// DispatchEvent dispatches event to gateway transformer for further processing
func (gc *GatewayConfig) DispatchEvent(event []byte, src string) error {
	payload, err := utils.TransformerPayload(event, src)
	if err != nil {
		gc.log.Warn().Str("config-key", src).Err(err).Msg("failed to transform request body.")
		return err
	} else {
		gc.log.Info().Str("config-key", src).Msg("dispatching the event to gateway-transformer...")

		_, err = http.Post(fmt.Sprintf("http://localhost:%s", gc.transformerPort), "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			gc.log.Warn().Str("config-key", src).Err(err).Msg("failed to dispatch event to gateway-transformer.")
			return err
		}
	}
	return nil
}

// creates an internal representation of configuration declared in the gateway configmap.
// returned configurations are map of hash of configuration and configuration itself.
// Creating a hash of configuration makes it easy to check equality of two configurations.
func (gc *GatewayConfig) createInternalConfigs(cm *corev1.ConfigMap) (map[uint64]*ConfigData, error) {
	configs := make(map[uint64]*ConfigData)
	for configKey, configValue := range cm.Data {
		hashKey, err := hs.Hash(configKey+configValue, &hs.HashOptions{})
		if err != nil {
			gc.log.Warn().Str("config-key", configKey).Str("config-value", configValue).Err(err).Msg("failed to hash configuration")
			return nil, err
		}

		gc.log.Info().Str("config-key", configKey).Interface("config-data", configValue).Str("hash", string(hashKey)).Msg("configuration hash")

		configs[hashKey] = &ConfigData{
			Src:    configKey,
			Config: configValue,
			StopCh: make(chan struct{}),
		}
	}
	return configs, nil
}

// diffConfig diffs currently registered configurations and the configurations in the gateway configmap
// retunrs staleConfig - configurations to be removed from gateway
// newConfig - new configurations to run
func (gc *GatewayConfig) diffConfigurations(newConfigs map[uint64]*ConfigData) (staleConfigKeys []uint64, newConfigKeys []uint64) {
	var currentConfigKeys []uint64
	var updatedConfigKeys []uint64

	for currentConfigKey, _ := range gc.registeredConfigs {
		currentConfigKeys = append(currentConfigKeys, currentConfigKey)
	}

	for updatedConfigKey, _ := range newConfigs {
		updatedConfigKeys = append(updatedConfigKeys, updatedConfigKey)
	}

	gc.log.Debug().Interface("current-config-keys", currentConfigKeys).Msg("hashes")
	gc.log.Debug().Interface("updated-config-keys", updatedConfigKeys).Msg("hashes")

	swapped := false
	// iterates over current configurations and updated configurations
	// and creates two arrays, first one containing configurations that need to removed
	// and second containing new configurations that need to be added and run.
	for i := 0; i < 2; i++ {
		for _, cc := range currentConfigKeys {
			found := false
			for _, uc := range updatedConfigKeys {
				if cc == uc {
					found = true
					break
				}
			}
			if !found {
				if swapped {
					newConfigKeys = append(newConfigKeys, cc)
				} else {
					staleConfigKeys = append(staleConfigKeys, cc)
				}
			}
		}
		if i == 0 {
			currentConfigKeys, updatedConfigKeys = updatedConfigKeys, currentConfigKeys
			swapped = true
		}
	}
	return
}

// NewGatewayConfiguration returns a new gateway configuration
func NewGatewayConfiguration() *GatewayConfig {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	name, _ := os.LookupEnv(common.GatewayName)
	if name == "" {
		panic("gateway name not provided")
	}

	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}

	transformerPort, ok := os.LookupEnv(common.GatewayTransformerPortEnvVar)
	if !ok {
		panic("gateway transformer port is not provided")
	}

	configName, ok := os.LookupEnv(common.GatewayProcessorConfigMapEnvVar)
	if !ok {
		panic("gateway processor configmap is not provided")
	}

	clientset := kubernetes.NewForConfigOrDie(restConfig)
	log := zlog.New(os.Stdout).With().Str("gateway-name", name).Logger()

	return &GatewayConfig{
		log:               log,
		registeredConfigs: make(map[uint64]*ConfigData),
		Clientset:         clientset,
		Namespace:         namespace,
		transformerPort:   transformerPort,
		configName:        configName,
	}
}
