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

package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	gwProto "github.com/argoproj/argo-events/gateways/grpc/proto"
	"github.com/argoproj/argo-events/gateways/utils"
	hs "github.com/mitchellh/hashstructure"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog"
	"google.golang.org/grpc"
	"io"
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

const (
	serverAddr = "localhost:%s"
)

// gatewayConfig provides a generic configuration for a gateway
type gatewayConfig struct {
	// Log provides fast and simple logger dedicated to JSON output
	log zerolog.Logger
	// Clientset is client for kubernetes API
	clientset *kubernetes.Clientset
	// Namespace is namespace for the gateway to run inside
	namespace string
	// TransformerPort is gateway transformer port to dispatch event to
	transformerPort string
	// registeredConfigs stores information about current configurations that are running in the gateway
	registeredConfigs map[uint64]*configData
	// rpcPort to run gRPC server on
	rpcPort string
}

type configData struct {
	// src is name of the configuration
	src string
	// config holds gateway configuration to run
	config string
	// cancel is called to cancel the context used by client to communicate with server.
	cancel context.CancelFunc
}

// Watches change in configuration for the gateway
func (gc *gatewayConfig) WatchGatewayConfigMap(ctx context.Context, name string) (cache.Controller, error) {
	source := gc.newConfigMapWatch(name)
	_, controller := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*corev1.ConfigMap); ok {
					gc.log.Info().Str("config-map", name).Msg("detected ConfigMap addition. Updating the controller config.")
					err := gc.manageConfigurations(cm)
					if err != nil {
						gc.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*corev1.ConfigMap); ok {
					gc.log.Info().Msg("detected ConfigMap update. Updating the controller config.")
					err := gc.manageConfigurations(newCm)
					if err != nil {
						gc.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// creates a new configmap watch
func (gc *gatewayConfig) newConfigMapWatch(name string) *cache.ListWatch {
	x := gc.clientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// syncs registered configurations and updated gateway configmap
func (gc *gatewayConfig) manageConfigurations(cm *corev1.ConfigMap) error {
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
		go gc.runConfig(newConfigKey, newConfig)
	}

	gc.log.Info().Interface("registered-configs", gc.registeredConfigs).Msg("registered configs")

	// remove stale configurations
	for _, staleConfigKey := range staleConfigKeys {
		staleConfig := gc.registeredConfigs[staleConfigKey]
		gc.log.Info().Str("config", staleConfig.src).Msg("deleting config")
		staleConfig.cancel()
		delete(gc.registeredConfigs, staleConfigKey)
	}
	return nil
}

// runConfig establishes connection with gateway server and sends new configurations to run.
// also disconnect the clients for stale configurations
func (gc *gatewayConfig) runConfig(hash uint64, config *configData) error {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf(serverAddr, gc.rpcPort), opts...)
	if err != nil {
		gc.log.Fatal().Str("config-key", config.src).Err(err).Msg("failed to dial")
	}

	// create a client for gateway server
	client := gwProto.NewGatewayExecutorClient(conn)

	ctx, cancel := context.WithCancel(context.Background())

	eventStream, err := client.RunGateway(ctx, &gwProto.GatewayConfig{
		Config: config.config,
		Src:    config.src,
	})

	gc.log.Info().Str("config-key", config.src).Msg("connected with server and got a event stream")

	if err != nil {
		gc.log.Error().Str("config-key", config.src).Err(err).Msg("failed to get event stream")
		return err
	}

	config.cancel = cancel

	for {
		event, err := eventStream.Recv()
		if err == io.EOF {
			gc.log.Info().Str("config-key", config.src).Msg("event stream stopped")
			return err
		}
		if err != nil {
			gc.log.Warn().Str("config-key", config.src).Err(err).Msg("failed to receive events on stream")
			return err
		}
		// event should never be nil
		if event == nil {
			return nil
		}
		payload, err := utils.TransformerPayload(event.Data, config.src)
		if err != nil {
			gc.log.Error().Str("config-key", config.src).Err(err).Msg("failed to transform request body")
			return err
		}

		gc.log.Info().Str("config-key", config.src).Msg("dispatching the event to gateway-transformer...")
		_, err = http.Post(fmt.Sprintf("http://localhost:%s", gc.transformerPort), "application/octet-stream", bytes.NewReader(payload))
	}
	return nil
}

// creates an internal representation of configuration declared in the gateway configmap.
// returned configurations are map of hash of configuration and configuration itself.
// Creating a hash of configuration makes it easy to check equality of two configurations.
func (gc *gatewayConfig) createInternalConfigs(cm *corev1.ConfigMap) (map[uint64]*configData, error) {
	configs := make(map[uint64]*configData)
	for configKey, configValue := range cm.Data {
		hasKey, err := hs.Hash(configValue, &hs.HashOptions{})
		if err != nil {
			gc.log.Warn().Str("config-key", configKey).Str("config-value", configValue).Err(err).Msg("failed to hash configuration")
			return nil, err
		}
		configs[hasKey] = &configData{
			src:    configKey,
			config: configValue,
		}
	}
	return configs, nil
}

// diffConfig diffs currently registered configurations and the configurations in the gateway configmap
// retunrs staleConfig - configurations to be removed from gateway
// newConfig - new configurations to run
func (gc *gatewayConfig) diffConfigurations(newConfigs map[uint64]*configData) (staleConfigKeys []uint64, newConfigKeys []uint64) {
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

func main() {
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

	rpcServerPort, ok := os.LookupEnv(common.GatewayProcessorServerPort)
	if !ok {
		panic("gateway rpc server port is not provided")
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

	gc := &gatewayConfig{
		log:               log,
		registeredConfigs: make(map[uint64]*configData),
		clientset:         clientset,
		namespace:         namespace,
		transformerPort:   transformerPort,
		rpcPort:           rpcServerPort,
	}

	gc.WatchGatewayConfigMap(context.Background(), configName)

	select {}
}
