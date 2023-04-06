package cmd

import (
	"context"
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
	"github.com/fsnotify/fsnotify"
)

func Start() {
	logger := logging.NewArgoEventsLogger().Named("sensor")

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

	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		logger.Fatalw("failed to get kubeconfig", zap.Error(err))
	}
	dynamicClient := dynamic.NewForConfigOrDie(restConfig)
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)

	ctx := logging.WithLogger(signals.SetupSignalHandler(), logger)

	manager := NewSensorContextManager(logger, kubeClient, dynamicClient, busConfig, ebSubject, hostname)
	defer manager.Close()

	logger.Infow("starting sensor manager", "version", argoevents.GetVersion())
	if err := manager.Start(ctx); err != nil {
		logger.Fatalw("failed to start sensor context manager", zap.Error(err))
	}

	for {
		select {
		case f := <-manager.C:
			if err := f(); err != nil {
				logger.Fatalw("failed to listen to events", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

type SensorContextManager struct {
	C               chan func() error
	logger          *zap.SugaredLogger
	kubeClient      kubernetes.Interface
	dynamicClient   dynamic.Interface
	eventBusConfig  *eventbusv1alpha1.BusConfig
	eventBusSubject string
	hostname        string
}

func NewSensorContextManager(logger *zap.SugaredLogger, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, eventBusConfig *eventbusv1alpha1.BusConfig, eventBusSubject, hostname string) *SensorContextManager {
	return &SensorContextManager{
		C:               make(chan func() error),
		logger:          logger,
		kubeClient:      kubeClient,
		dynamicClient:   dynamicClient,
		eventBusConfig:  eventBusConfig,
		eventBusSubject: eventBusSubject,
		hostname:        hostname,
	}
}

func (scm *SensorContextManager) Start(ctx context.Context) error {
	// This env var is used to indicate if this sensor definition has opted in
	// to live reload
	path, liveReload := os.LookupEnv(common.EnvVarSensorFilePath)

	switch liveReload {
	case true:
		scm.logger.Infow("starting live reload sensor")
		return scm.startLiveReload(ctx, path)
	case false:
		scm.logger.Infow("starting default sensor")
		return scm.startDefault(ctx)
	}

	return nil
}

func (scm *SensorContextManager) Close() {
	close(scm.C)
}

func (scm *SensorContextManager) startDefault(ctx context.Context) error {
	sensor, err := scm.readSensorFromEnv()
	if err != nil {
		scm.logger.Fatalw("failed to read sensor from env variable", zap.Error(err))
	}

	m := metrics.NewMetrics(sensor.Namespace)
	go m.Run(ctx, fmt.Sprintf(":%d", common.SensorMetricsPort))
	go scm.newContext(ctx, sensor, m)

	return nil
}

func (scm *SensorContextManager) startLiveReload(ctx context.Context, path string) error {
	// watch for sensor file changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.Add(path)
	if err != nil {
		return err
	}

	sensor, err := scm.readSensorFromFile(path)
	if err != nil {
		scm.logger.Fatalw("failed to read sensor from file", zap.Error(err))
	}

	// start metrics server once
	m := metrics.NewMetrics(sensor.Namespace)
	go m.Run(ctx, fmt.Sprintf(":%d", common.SensorMetricsPort))

	go func() {
		defer watcher.Close()

		// initial sensor context
		subCtx, cancel := context.WithCancel(ctx)
		scm.newContext(subCtx, sensor, m)

		// continuously listen for file (sensor) updates
		for {
			select {
			case event := <-watcher.Events:
				// The kubelet will atomically replace the sensor file (that
				// is volume mounted based on a configmap), when this happens
				// the file is removed and created, so we need to look for
				// both write and remove events.
				if event.Op == fsnotify.Write || event.Op == fsnotify.Remove {
					cancel()

					sensor, err := scm.readSensorFromFile(path)
					if err != nil {
						scm.logger.Fatalw("failed to read sensor from file", zap.Error(err))
					}

					// refreshed sensor context
					subCtx, cancel = context.WithCancel(ctx)
					scm.newContext(subCtx, sensor, m)

					if err := watcher.Add(path); err != nil {
						scm.logger.Fatalw("failed to add sensor watch", zap.Error(err))
					}
				}
			case <-ctx.Done():
				cancel()
				return
			}
		}
	}()

	return nil
}

func (scm *SensorContextManager) readSensorFromEnv() (*v1alpha1.Sensor, error) {
	encodedSensorSpec, defined := os.LookupEnv(common.EnvVarSensorObject)
	if !defined {
		return nil, fmt.Errorf("required environment variable '%s' not defined", common.EnvVarSensorObject)
	}
	sensorSpec, err := base64.StdEncoding.DecodeString(encodedSensorSpec)
	if err != nil {
		return nil, err
	}
	sensor := &v1alpha1.Sensor{}
	if err = json.Unmarshal(sensorSpec, sensor); err != nil {
		return nil, err
	}

	// logger
	scm.logger = scm.logger.With("sensorName", sensor.Name)
	for name, value := range sensor.Spec.LoggingFields {
		scm.logger.With(name, value)
	}

	return scm.verifySensor(sensor), nil
}

func (scm *SensorContextManager) readSensorFromFile(path string) (*v1alpha1.Sensor, error) {
	sensorSpec, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	sensor := &v1alpha1.Sensor{}
	if err = json.Unmarshal(sensorSpec, sensor); err != nil {
		return nil, err
	}

	// logger
	scm.logger = scm.logger.With("sensorName", sensor.Name)
	for name, value := range sensor.Spec.LoggingFields {
		scm.logger.With(name, value)
	}

	return scm.verifySensor(sensor), nil
}

func (scm *SensorContextManager) verifySensor(sensor *v1alpha1.Sensor) *v1alpha1.Sensor {
	if scm.eventBusConfig.NATS != nil {
		for _, trigger := range sensor.Spec.Triggers {
			if trigger.AtLeastOnce {
				scm.logger.Warn("ignoring atLeastOnce when using NATS")
				trigger.AtLeastOnce = false
			}
		}
	}

	return sensor
}

func (scm *SensorContextManager) newContext(ctx context.Context, sensor *v1alpha1.Sensor, m *metrics.Metrics) {
	sensorCtx := sensors.NewSensorContext(scm.kubeClient, scm.dynamicClient, sensor, scm.eventBusConfig, scm.eventBusSubject, scm.hostname, m)
	scm.C <- func() error { return sensorCtx.Start(ctx) }
}
