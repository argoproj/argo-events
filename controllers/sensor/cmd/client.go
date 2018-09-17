package main

import (
	"github.com/argoproj/argo-events/common"
	sc "github.com/argoproj/argo-events/controllers/sensor"
	sv1 "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/discovery"
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

	// initialize logger
	log := zerolog.New(os.Stdout).With().Str("sensor-name", sensorName).Logger()
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
	sensorExecutionCtx := sc.NewsensorExecutionCtx(sensorClient, kubeClient, clientPool, disco, sensor, log, &wg)
	go sensorExecutionCtx.WatchSignalNotifications()
	wg.Wait()
}
