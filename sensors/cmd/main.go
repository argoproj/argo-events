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
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	encodedSensorSpec, defined := os.LookupEnv(common.EnvVarSensorObject)
	if !defined {
		panic(errors.Errorf("required environment variable '%s' not defined", common.EnvVarSensorObject))
	}
	sensorSpec, err := base64.StdEncoding.DecodeString(encodedSensorSpec)
	if err != nil {
		panic(errors.Errorf("failed to decode sensor string. err: %+v", err))
	}
	sensor := &v1alpha1.Sensor{}
	if err = json.Unmarshal([]byte(sensorSpec), sensor); err != nil {
		panic(errors.Errorf("failed to unmarshal sensor object. err: %+v", err))
	}

	busConfig := &eventbusv1alpha1.BusConfig{}
	encodedBusConfigSpec := os.Getenv(common.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			panic(errors.Errorf("failed to decode bus config string. err: %+v", err))
		}
		if err = json.Unmarshal([]byte(busConfigSpec), busConfig); err != nil {
			panic(errors.Errorf("failed to unmarshal bus config object. err: %+v", err))
		}
	}

	ebSubject, defined := os.LookupEnv(common.EnvVarEventBusSubject)
	if !defined {
		panic(errors.Errorf("required environment variable '%s' not defined", common.EnvVarEventBusSubject))
	}

	dynamicClient := dynamic.NewForConfigOrDie(restConfig)

	sensorExecutionCtx := sensors.NewSensorContext(kubeClient, dynamicClient, sensor, busConfig, ebSubject)
	if err := sensorExecutionCtx.ListenEvents(); err != nil {
		sensorExecutionCtx.Logger.WithError(err).Errorln("failed to listen to events")
		os.Exit(-1)
	}
}
