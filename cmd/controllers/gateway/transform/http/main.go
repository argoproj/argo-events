package main

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers/gateway/transform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"os"
	"strings"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
)

func main() {
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)

	namespace, _ := os.LookupEnv(common.EnvVarNamespace)
	if namespace == "" {
		panic("no namespace provided")
	}

	configmap, _ := os.LookupEnv(common.GatewayTransformerConfigMapEnvVar)
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

	fmt.Sprintf("sensor watchers: %s", sensorWatchersStr)
	fmt.Sprintf("gateway watchers: %s", gatewayWatchersStr)

	if sensorWatchersStr != "" {
		for _, sensorWatcherStr := range strings.Split(sensorWatchersStr, ",") {
			fmt.Sprintf("unmarshalling sensor watcher: %s", sensorWatcherStr)
			var sensorWatcher v1alpha1.SensorNotificationWatcher
			err = yaml.Unmarshal([]byte(sensorWatcherStr), &sensorWatcher)
			if err != nil {
				panic(fmt.Errorf("failed to unmarshal sensor watcher string. Err: %+v", err))
			}
			sensorWatchers = append(sensorWatchers, sensorWatcher)
		}
	}

	if gatewayWatchersStr != "" {
		for _, gatewayWatcherStr := range strings.Split(gatewayWatchersStr, ",") {
			fmt.Sprintf("unmarshalling gateway watcher: %s", gatewayWatcherStr)
			var gatewayWatcher v1alpha1.GatewayNotificationWatcher
			err = yaml.Unmarshal([]byte(gatewayWatcherStr), &gatewayWatcher)
			if err != nil {
				panic(fmt.Errorf("failed to unmarshal gateway watcher string. Err: %+v", err))
			}
			gatewayWatchers = append(gatewayWatchers, gatewayWatcher)
		}
	}

	transformerConfig := transform.NewTransformerConfig(tConfigMapData[common.EventType],
		tConfigMapData[common.EventTypeVersion],
		tConfigMapData[common.EventSource],
		sensorWatchers,
		gatewayWatchers,
			)

	// Create an operation context
	toc := transform.NewTransformOperationContext(transformerConfig, namespace, kubeClient)
	ctx := context.Background()
	_, err = toc.WatchGatewayTransformerConfigMap(ctx, configmap)
	if err != nil {
		log.Fatalf("failed to register watch for store config map: %+v", err)
	}

	http.HandleFunc("/", toc.TransformRequest)
	log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%s", common.GatewayTransformerPort), nil))
}
