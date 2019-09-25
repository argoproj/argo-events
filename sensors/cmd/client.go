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
	"fmt"
	"github.com/argoproj/argo-events/common"
	sv1 "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	sc "github.com/argoproj/argo-events/sensors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	dynamic "k8s.io/client-go/deprecated-dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
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

	sensorClient, err := sv1.NewForConfig(restConfig)
	if err != nil {
		panic(fmt.Errorf("failed to get sensor client. err: %+v", err))
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	sensor, err := sensorClient.ArgoprojV1alpha1().Sensors(sensorNamespace).Get(sensorName, metav1.GetOptions{})
	if err != nil {
		panic(fmt.Errorf("failed to retrieve sensor. err: %+v", err))
	}

	clientPool := dynamic.NewDynamicClientPool(restConfig)
	disco := discovery.NewDiscoveryClientForConfigOrDie(restConfig)

	// wait for sensor http server to shutdown
	sensorExecutionCtx := sc.NewSensorExecutionCtx(sensorClient, kubeClient, clientPool, disco, sensor, controllerInstanceID)
	sensorExecutionCtx.WatchEventsFromGateways()
}
