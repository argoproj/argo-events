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
	"github.com/argoproj/argo-events/common"
	sc "github.com/argoproj/argo-events/controllers/sensor"
	sv1 "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	"sync"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	sensorName, ok := os.LookupEnv(common.SensorName)
	if !ok {
		panic("sensor name is not provided")
	}
	sensorNamespace, ok := os.LookupEnv(common.SensorNamespace)
	if !ok {
		panic("sensor namespace is not provided")
	}
	controllerInstanceID, ok := os.LookupEnv(common.EnvVarSensorControllerInstanceID)
	if !ok {
		panic("sensor controller instance ID is not provided")
	}

	// initialize logger
	log := zerolog.New(os.Stdout).With().Str("sensor-name", sensorName).Caller().Logger()
	sensorClient, err := sv1.NewForConfig(restConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get sensor client")
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	sensor, err := sensorClient.ArgoprojV1alpha1().Sensors(sensorNamespace).Get(sensorName, metav1.GetOptions{})
	if err != nil {
		log.Panic().Err(err).Msg("failed to retrieve sensor")
	}

	clientPool := dynamic.NewDynamicClientPool(restConfig)
	disco := discovery.NewDiscoveryClientForConfigOrDie(restConfig)

	// wait for sensor http server to shutdown
	var wg sync.WaitGroup
	wg.Add(1)
	sensorExecutionCtx := sc.NewsensorExecutionCtx(sensorClient, kubeClient, clientPool, disco, sensor, log, &wg, controllerInstanceID)
	go sensorExecutionCtx.WatchSignalNotifications()
	wg.Wait()
}
