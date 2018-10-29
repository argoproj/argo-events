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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers/gateway/transform"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)

	namespace, _ := os.LookupEnv(common.GatewayNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}

	configmap, _ := os.LookupEnv(common.EnvVarGatewayTransformerConfigMap)
	if configmap == "" {
		panic("no gateway transformer config-map provided.")
	}

	tConfigMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(configmap, metav1.GetOptions{})

	if err != nil {
		panic(fmt.Errorf("failed to retrieve config map. Err: %+v", err))
	}

	// create the configuration for gateway transformer
	tConfigMapData := tConfigMap.Data
	// parse sensor watchers and gateway watchers if any
	var sensorWatchers []v1alpha1.SensorNotificationWatcher
	var gatewayWatchers []v1alpha1.GatewayNotificationWatcher

	sensorWatchersStr := tConfigMapData[common.SensorWatchers]
	gatewayWatchersStr := tConfigMapData[common.GatewayWatchers]

	// parse sensor and gateway watchers for this gateway
	if sensorWatchersStr != "" {
		for _, sensorWatcherStr := range strings.Split(sensorWatchersStr, ",") {
			var sensorWatcher v1alpha1.SensorNotificationWatcher
			err = yaml.Unmarshal([]byte(sensorWatcherStr), &sensorWatcher)
			if err != nil {
				panic(fmt.Errorf("failed to parse sensor watcher string. Err: %+v", err))
			}
			sensorWatchers = append(sensorWatchers, sensorWatcher)
		}
	}

	if gatewayWatchersStr != "" {
		for _, gatewayWatcherStr := range strings.Split(gatewayWatchersStr, ",") {
			var gatewayWatcher v1alpha1.GatewayNotificationWatcher
			err = yaml.Unmarshal([]byte(gatewayWatcherStr), &gatewayWatcher)
			if err != nil {
				panic(fmt.Errorf("failed to unmarshal gateway watcher string. Err: %+v", err))
			}
			gatewayWatchers = append(gatewayWatchers, gatewayWatcher)
		}
	}

	// creates a new configuration for gateway transformer. Gateway transformer is responsible for
	// converting events received from gateway processor into cloudevents specification compliant events
	// and dispatch them to watchers(components interested in listening to events produced by this gateway)
	transformerConfig := transform.NewTransformerConfig(tConfigMapData[common.EventType],
		tConfigMapData[common.EventTypeVersion],
		tConfigMapData[common.EventSource],
		sensorWatchers,
		gatewayWatchers,
	)

	// Create an operation context for transformer
	toc := transform.NewTransformOperationContext(transformerConfig, namespace, kubeClient)
	ctx := context.Background()

	// configmap for gateway transformer contains information necessary to convert an event received from
	// gateway processor into cloudevents specification compliant event.
	_, err = toc.WatchGatewayTransformerConfigMap(ctx, configmap)
	if err != nil {
		log.Fatalf("failed to register watch for store config map: %+v", err)
	}

	// endpoint to listen events
	http.HandleFunc("/", toc.TransformRequest)
	log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%s", common.GatewayTransformerPort), nil))
}
