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
	"log"
	"os"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	"github.com/blackrock/axis/common"
	"github.com/blackrock/axis/controller"
	sensorclientset "github.com/blackrock/axis/pkg/client/clientset/versioned"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		log.Fatal(err)
	}

	kubeClientset := kubernetes.NewForConfigOrDie(restConfig)
	sensorClientset := sensorclientset.NewForConfigOrDie(restConfig)

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err.Error())
	}

	configMap, ok := os.LookupEnv(common.EnvVarConfigMap)
	if !ok {
		configMap = common.DefaultConfigMapName(common.DefaultSensorControllerDeploymentName)
	}

	controller := controller.NewSensorController(restConfig, kubeClientset, sensorClientset, logger.Sugar(), configMap)
	err = controller.ResyncConfig()
	if err != nil {
		log.Fatalf("%+v", err)
	}

	go controller.Run(context.Background(), 1, 1)

	// Wait forever
	select {}
}
