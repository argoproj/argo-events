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
	configMap, ok := os.LookupEnv(common.EnvVarControllerConfigmap)
	if !ok {
		configMap = common.DefaultConfigMapName(common.DefaultGatewayControllerDeploymentName)
	}

	namespace, ok := os.LookupEnv(common.EnvVarControllerNamespace)
	if !ok {
		namespace = common.DefaultControllerNamespace
	}

	// create new gateway controller
	controller := gateway.NewGatewayController(restConfig, configMap, namespace)
	// watch for configuration updates for the controller
	err = controller.SyncControllerConfig()
	if err != nil {
		panic(err)
	}

	go controller.Run(context.Background(), 1, 1)

	// Wait forever
	select {}
}
