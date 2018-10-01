package main

import (
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers/gateway"
	"os"
)

func main() {
	// kubernetes configuration
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// gateway-controller configuration
	configMap, ok := os.LookupEnv(common.GatewayControllerConfigMapEnvVar)
	if !ok {
		configMap = common.DefaultConfigMapName(common.DefaultGatewayControllerDeploymentName)
	}

	namespace, ok := os.LookupEnv(common.GatewayNamespace)
	if !ok {
		namespace = common.DefaultGatewayControllerNamespace
	}

	// create new gateway controller
	controller := gateway.NewGatewayController(restConfig, configMap)
	// watch for configuration updates for the controller
	err = controller.ResyncConfig(namespace)
	if err != nil {
		panic(err)
	}

	go controller.Run(context.Background(), 1, 1)

	// Wait forever
	select {}
}
