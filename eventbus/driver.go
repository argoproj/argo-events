package eventbus

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common/logging"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	jetstreamsource "github.com/argoproj/argo-events/eventbus/jetstream/eventsource"
	jetstreamsensor "github.com/argoproj/argo-events/eventbus/jetstream/sensor"
	kafkasource "github.com/argoproj/argo-events/eventbus/kafka/eventsource"
	stansource "github.com/argoproj/argo-events/eventbus/stan/eventsource"
	stansensor "github.com/argoproj/argo-events/eventbus/stan/sensor"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/spf13/viper"
)

func GetEventSourceDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, eventSourceName string, defaultSubject string) (eventbuscommon.EventSourceDriver, error) {
	auth, err := GetAuth(ctx, eventBusConfig)
	if err != nil {
		return nil, err
	}
	if eventSourceName == "" {
		return nil, fmt.Errorf("eventSourceName must be specified to create eventbus driver")
	}

	logger := logging.FromContext(ctx)

	logger.Infof("eventBusConfig: %+v", eventBusConfig)

	var eventBusType apicommon.EventBusType
	switch {
	case eventBusConfig.NATS != nil && eventBusConfig.JetStream != nil:
		return nil, fmt.Errorf("invalid event bus, NATS and Jetstream shouldn't both be specified")
	case eventBusConfig.NATS != nil:
		eventBusType = apicommon.EventBusNATS
	case eventBusConfig.JetStream != nil:
		eventBusType = apicommon.EventBusJetStream
	case eventBusConfig.Kafka != nil:
		eventBusType = apicommon.EventBusKafka
	default:
		return nil, fmt.Errorf("invalid event bus")
	}

	var dvr eventbuscommon.EventSourceDriver
	switch eventBusType {
	case apicommon.EventBusNATS:
		if defaultSubject == "" {
			return nil, fmt.Errorf("subject must be specified to create NATS Streaming driver")
		}
		dvr = stansource.NewSourceSTAN(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, eventSourceName, defaultSubject, auth, logger)
	case apicommon.EventBusJetStream:
		dvr, err = jetstreamsource.NewSourceJetstream(eventBusConfig.JetStream.URL, eventSourceName, eventBusConfig.JetStream.StreamConfig, auth, logger) // don't need to pass in subject because subjects will be derived from dependencies
		if err != nil {
			return nil, err
		}
	case apicommon.EventBusKafka:
		dvr = kafkasource.NewKafkaSource([]string{}, logger)
	default:
		return nil, fmt.Errorf("invalid eventbus type")
	}
	return dvr, nil
}

func GetSensorDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, sensorSpec *v1alpha1.Sensor) (eventbuscommon.SensorDriver, error) {
	auth, err := GetAuth(ctx, eventBusConfig)
	if err != nil {
		return nil, err
	}

	if sensorSpec == nil {
		return nil, fmt.Errorf("sensorSpec required for getting eventbus driver")
	}
	if sensorSpec.Name == "" {
		return nil, fmt.Errorf("sensorSpec name must be set for getting eventbus driver")
	}
	logger := logging.FromContext(ctx)

	var eventBusType apicommon.EventBusType
	switch {
	case eventBusConfig.NATS != nil && eventBusConfig.JetStream != nil:
		return nil, fmt.Errorf("invalid event bus, NATS and Jetstream shouldn't both be specified")
	case eventBusConfig.NATS != nil:
		eventBusType = apicommon.EventBusNATS
	case eventBusConfig.JetStream != nil:
		eventBusType = apicommon.EventBusJetStream
	default:
		return nil, fmt.Errorf("invalid event bus")
	}

	var dvr eventbuscommon.SensorDriver
	switch eventBusType {
	case apicommon.EventBusNATS:
		dvr = stansensor.NewSensorSTAN(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, sensorSpec.Name, auth, logger)
		return dvr, nil
	case apicommon.EventBusJetStream:
		dvr, err = jetstreamsensor.NewSensorJetstream(eventBusConfig.JetStream.URL, sensorSpec, eventBusConfig.JetStream.StreamConfig, auth, logger) // don't need to pass in subject because subjects will be derived from dependencies
		return dvr, err
	default:
		return nil, fmt.Errorf("invalid eventbus type")
	}
}

func GetAuth(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig) (*eventbuscommon.Auth, error) {
	logger := logging.FromContext(ctx)

	var eventBusAuth *eventbusv1alpha1.AuthStrategy
	switch {
	case eventBusConfig.NATS != nil:
		eventBusAuth = eventBusConfig.NATS.Auth
	case eventBusConfig.JetStream != nil:
		eventBusAuth = &eventbusv1alpha1.AuthStrategyBasic
	default:
		return nil, fmt.Errorf("invalid event bus")
	}
	var auth *eventbuscommon.Auth
	cred := &eventbuscommon.AuthCredential{}
	if eventBusAuth == nil || *eventBusAuth == eventbusv1alpha1.AuthStrategyNone {
		auth = &eventbuscommon.Auth{
			Strategy: eventbusv1alpha1.AuthStrategyNone,
		}
	} else {
		v := viper.New()
		v.SetConfigName("auth")
		v.SetConfigType("yaml")
		v.AddConfigPath(common.EventBusAuthFileMountPath)
		err := v.ReadInConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load auth.yaml. err: %w", err)
		}
		err = v.Unmarshal(cred)
		if err != nil {
			logger.Errorw("failed to unmarshal auth.yaml", zap.Error(err))
			return nil, err
		}
		v.WatchConfig()
		v.OnConfigChange(func(e fsnotify.Event) {
			// Auth file changed, let it restart
			logger.Fatal("Eventbus auth config file changed, exiting..")
		})
		auth = &eventbuscommon.Auth{
			Strategy:   *eventBusAuth,
			Credential: cred,
		}
	}

	return auth, nil
}
