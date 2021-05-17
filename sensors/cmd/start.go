package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	argoevents "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/metrics"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	v1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors"
)

func Start() {
	logger := logging.NewArgoEventsLogger().Named("sensor")
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		logger.Fatalw("failed to get kubeconfig", zap.Error(err))
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	encodedSensorSpec, defined := os.LookupEnv(common.EnvVarSensorObject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", common.EnvVarSensorObject)
	}
	sensorSpec, err := base64.StdEncoding.DecodeString(encodedSensorSpec)
	if err != nil {
		logger.Fatalw("failed to decode sensor string", zap.Error(err))
	}
	sensor := &v1alpha1.Sensor{}
	if err = json.Unmarshal(sensorSpec, sensor); err != nil {
		logger.Fatalw("failed to unmarshal sensor object", zap.Error(err))
	}

	busConfig := &eventbusv1alpha1.BusConfig{}
	encodedBusConfigSpec := os.Getenv(common.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			logger.Fatalw("failed to decode bus config string", zap.Error(err))
		}
		if err = json.Unmarshal(busConfigSpec, busConfig); err != nil {
			logger.Fatalw("failed to unmarshal bus config object", zap.Error(err))
		}
	}

	ebSubject, defined := os.LookupEnv(common.EnvVarEventBusSubject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", common.EnvVarEventBusSubject)
	}

	hostname, defined := os.LookupEnv("POD_NAME")
	if !defined {
		logger.Fatal("required environment variable 'POD_NAME' not defined")
	}

	dynamicClient := dynamic.NewForConfigOrDie(restConfig)

	logger = logger.With("sensorName", sensor.Name)
	ctx := logging.WithLogger(signals.SetupSignalHandler(), logger)
	m := metrics.NewMetrics(sensor.Namespace)
	go m.Run(ctx, fmt.Sprintf(":%d", common.SensorMetricsPort))

	logger.Infow("starting sensor server", "version", argoevents.GetVersion())
	sensorExecutionCtx := sensors.NewSensorContext(kubeClient, dynamicClient, sensor, busConfig, ebSubject, hostname, m)
	if err := sensorExecutionCtx.Start(ctx); err != nil {
		logger.Fatalw("failed to listen to events", zap.Error(err))
	}
}
