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
	"context"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controller"
	"github.com/argoproj/argo-events/shared"
)

func main() {
	// kubernetes configuration
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// controller configuration
	configMap, ok := os.LookupEnv(common.EnvVarConfigMap)
	if !ok {
		configMap = common.DefaultConfigMapName(common.DefaultSensorControllerDeploymentName)
	}

	// logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	// stream signal plugins
	pluginFile := os.Getenv("STREAM_PLUGIN")
	pluginClient := initPlugins(pluginFile)
	defer pluginClient.Kill()

	streamClient, err := pluginClient.Client()
	if err != nil {
		panic(err)
	}

	controller := controller.NewSensorController(restConfig, configMap, streamClient, logger.Sugar())
	err = controller.ResyncConfig()
	if err != nil {
		panic(err)
	}

	go controller.Run(context.Background(), 1, 1)

	// Wait forever
	select {}
}

func initPlugins(pluginFile string) *plugin.Client {
	return plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(pluginFile),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC, plugin.ProtocolGRPC,
		},
	})
}
