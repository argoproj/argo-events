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
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/sensors"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

func Start() {
	logger := logging.NewArgoEventsLogger().Named("sensor")
	kubeConfig, _ := os.LookupEnv(v1alpha1.EnvVarKubeConfig)
	restConfig, err := sharedutil.GetClientConfig(kubeConfig)
	if err != nil {
		logger.Fatalw("failed to get kubeconfig", zap.Error(err))
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	encodedSensorSpec, defined := os.LookupEnv(v1alpha1.EnvVarSensorObject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", v1alpha1.EnvVarSensorObject)
	}
	sensorSpec, err := base64.StdEncoding.DecodeString(encodedSensorSpec)
	if err != nil {
		logger.Fatalw("failed to decode sensor string", zap.Error(err))
	}
	sensor := &v1alpha1.Sensor{}
	if err = json.Unmarshal(sensorSpec, sensor); err != nil {
		logger.Fatalw("failed to unmarshal sensor object", zap.Error(err))
	}

	busConfig := &v1alpha1.BusConfig{}
	encodedBusConfigSpec := os.Getenv(v1alpha1.EnvVarEventBusConfig)
	if len(encodedBusConfigSpec) > 0 {
		busConfigSpec, err := base64.StdEncoding.DecodeString(encodedBusConfigSpec)
		if err != nil {
			logger.Fatalw("failed to decode bus config string", zap.Error(err))
		}
		if err = json.Unmarshal(busConfigSpec, busConfig); err != nil {
			logger.Fatalw("failed to unmarshal bus config object", zap.Error(err))
		}
	}
	if busConfig.NATS != nil {
		for _, trigger := range sensor.Spec.Triggers {
			if trigger.AtLeastOnce {
				logger.Warn("ignoring atLeastOnce when using NATS")
				trigger.AtLeastOnce = false
			}
		}
	}
	ebSubject, defined := os.LookupEnv(v1alpha1.EnvVarEventBusSubject)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", v1alpha1.EnvVarEventBusSubject)
	}

	hostname, defined := os.LookupEnv("POD_NAME")
	if !defined {
		logger.Fatal("required environment variable 'POD_NAME' not defined")
	}

	dynamicClient := dynamic.NewForConfigOrDie(restConfig)

	logger = logger.With("sensorName", sensor.Name)
	for name, value := range sensor.Spec.LoggingFields {
		logger.With(name, value)
	}

	ctx := logging.WithLogger(signals.SetupSignalHandler(), logger)
	m := metrics.NewMetrics(sensor.Namespace)
	go m.Run(ctx, fmt.Sprintf(":%d", v1alpha1.SensorMetricsPort))

	logger.Infow("starting sensor server", "version", argoevents.GetVersion())
	sensorExecutionCtx := sensors.NewSensorContext(kubeClient, dynamicClient, sensor, busConfig, ebSubject, hostname, m)
	if err := sensorExecutionCtx.Start(ctx); err != nil {
		logger.Fatalw("failed to listen to events", zap.Error(err))
	}
}
