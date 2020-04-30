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
	"fmt"
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers/gateway"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
)

func main() {
	// kubernetes configuration
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// gateway-controller configuration
	configMap, ok := os.LookupEnv(common.EnvVarControllerConfigMap)
	if !ok {
		panic("controller configmap is not provided")
	}

	namespace, ok := os.LookupEnv(common.EnvVarNamespace)
	if !ok {
		namespace = common.DefaultControllerNamespace
	}

	clientImage, ok := os.LookupEnv(common.EnvVarClientImage)
	if !ok {
		panic(fmt.Sprintf("Env var %s for gateway client image is not provided", common.EnvVarClientImage))
	}
	gatewayImageRegistry, ok := os.LookupEnv(common.EnvVarImageRegistry)
	if !ok {
		gatewayImageRegistry = "docker.io"
	}
	gatewayImageVersion, ok := os.LookupEnv(common.EnvVarImageVersion)
	if !ok {
		gatewayImageVersion = "latest"
	}
	gatewayImages := getGatewayImages(gatewayImageRegistry, gatewayImageVersion)
	// create new gateway controller
	controller := gateway.NewGatewayController(restConfig, configMap, namespace, clientImage, gatewayImages)
	// watch for configuration updates for the controller
	err = controller.ResyncConfig(namespace)
	if err != nil {
		panic(err)
	}

	go controller.Run(context.Background(), 1)
	select {}
}

// getGatewayImages is used to get the gateway image mapping
// TODO: temporarily solution, need to figure out a better way
func getGatewayImages(gatewayImageRegistry, gatewayImageVersion string) map[apicommon.EventSourceType]string {
	var result = make(map[apicommon.EventSourceType]string)
	for k, v := range apicommon.GetGatewayImageNameMapping() {
		result[k] = fmt.Sprintf("%s/%s:%s", gatewayImageRegistry, v, gatewayImageVersion)
	}
	return result
}
