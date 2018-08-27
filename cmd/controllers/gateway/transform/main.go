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
	transformerConfig := transform.NewTransformerConfig(tConfigMapData[common.EventType], tConfigMapData[common.EventTypeVersion], tConfigMapData[common.EventSource], strings.Split(tConfigMapData[common.SensorList], ","))

	// Create an operation context
	eoc := transform.NewTransformOperationContext(transformerConfig, namespace, kubeClient)
	ctx := context.Background()
	_, err = eoc.WatchGatewayTransformerConfigMap(ctx, configmap)
	if err != nil {
		log.Fatalf("failed to register watch for store config map: %+v", err)
	}

	http.HandleFunc("/", eoc.TransformRequest)
	log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%d", common.GatewayTransformerPort), nil))
}
