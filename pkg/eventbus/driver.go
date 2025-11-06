package eventbus

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	jetstreamsource "github.com/argoproj/argo-events/pkg/eventbus/jetstream/eventsource"
	jetstreamsensor "github.com/argoproj/argo-events/pkg/eventbus/jetstream/sensor"
	kafkasource "github.com/argoproj/argo-events/pkg/eventbus/kafka/eventsource"
	kafkasensor "github.com/argoproj/argo-events/pkg/eventbus/kafka/sensor"
	stansource "github.com/argoproj/argo-events/pkg/eventbus/stan/eventsource"
	stansensor "github.com/argoproj/argo-events/pkg/eventbus/stan/sensor"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

func GetEventSourceDriver(ctx context.Context, eventBusConfig v1alpha1.BusConfig, eventSourceName string, defaultSubject string) (eventbuscommon.EventSourceDriver, error) {
	auth, err := GetAuth(ctx, eventBusConfig)
	if err != nil {
		return nil, err
	}
	if eventSourceName == "" {
		return nil, fmt.Errorf("eventSourceName must be specified to create eventbus driver")
	}

	logger := logging.FromContext(ctx)

	logger.Infof("eventBusConfig: %+v", eventBusConfig)

	var eventBusType v1alpha1.EventBusType
	switch {
	case eventBusConfig.NATS != nil:
		eventBusType = v1alpha1.EventBusNATS
	case eventBusConfig.JetStream != nil:
		eventBusType = v1alpha1.EventBusJetStream
	case eventBusConfig.Kafka != nil:
		eventBusType = v1alpha1.EventBusKafka
	default:
		return nil, fmt.Errorf("invalid event bus")
	}

	var dvr eventbuscommon.EventSourceDriver
	switch eventBusType {
	case v1alpha1.EventBusNATS:
		if defaultSubject == "" {
			return nil, fmt.Errorf("subject must be specified to create NATS Streaming driver")
		}
		dvr = stansource.NewSourceSTAN(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, eventSourceName, defaultSubject, auth, logger)
	case v1alpha1.EventBusJetStream:
		dvr, err = jetstreamsource.NewSourceJetstream(eventBusConfig.JetStream.URL, eventSourceName, eventBusConfig.JetStream.StreamConfig, auth, logger, eventBusConfig.JetStream.TLS) // don't need to pass in subject because subjects will be derived from dependencies
		if err != nil {
			return nil, err
		}
	case v1alpha1.EventBusKafka:
		dvr = kafkasource.NewKafkaSource(eventBusConfig.Kafka, logger)
	default:
		return nil, fmt.Errorf("invalid eventbus type")
	}
	return dvr, nil
}

func GetSensorDriver(ctx context.Context, eventBusConfig v1alpha1.BusConfig, sensorSpec *v1alpha1.Sensor, hostname string) (eventbuscommon.SensorDriver, error) {
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

	var eventBusType v1alpha1.EventBusType
	switch {
	case eventBusConfig.NATS != nil:
		eventBusType = v1alpha1.EventBusNATS
	case eventBusConfig.JetStream != nil:
		eventBusType = v1alpha1.EventBusJetStream
	case eventBusConfig.Kafka != nil:
		eventBusType = v1alpha1.EventBusKafka
	default:
		return nil, fmt.Errorf("invalid event bus")
	}

	var dvr eventbuscommon.SensorDriver
	switch eventBusType {
	case v1alpha1.EventBusNATS:
		dvr = stansensor.NewSensorSTAN(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, sensorSpec.Name, auth, logger)
		return dvr, nil
	case v1alpha1.EventBusJetStream:
		dvr, err = jetstreamsensor.NewSensorJetstream(eventBusConfig.JetStream.URL, sensorSpec, eventBusConfig.JetStream.StreamConfig, auth, logger, eventBusConfig.JetStream.TLS) // don't need to pass in subject because subjects will be derived from dependencies
		return dvr, err
	case v1alpha1.EventBusKafka:
		dvr = kafkasensor.NewKafkaSensor(eventBusConfig.Kafka, sensorSpec, hostname, logger)
		return dvr, nil
	default:
		return nil, fmt.Errorf("invalid eventbus type")
	}
}

func GetAuth(ctx context.Context, eventBusConfig v1alpha1.BusConfig) (*eventbuscommon.Auth, error) {
	logger := logging.FromContext(ctx)

	var eventBusAuth *v1alpha1.AuthStrategy
	switch {
	case eventBusConfig.NATS != nil:
		eventBusAuth = eventBusConfig.NATS.Auth
	case eventBusConfig.JetStream != nil:
		switch {
		case eventBusConfig.JetStream.Auth != nil:
			eventBusAuth = eventBusConfig.JetStream.Auth
		case eventBusConfig.JetStream.AccessSecret != nil:
			// For backward compatibility, default to Basic auth if AccessSecret is present but Auth is not specified
			eventBusAuth = &v1alpha1.AuthStrategyBasic
		default:
			eventBusAuth = nil
		}
	case eventBusConfig.Kafka != nil:
		eventBusAuth = nil
	default:
		return nil, fmt.Errorf("invalid event bus")
	}
	var auth *eventbuscommon.Auth
	cred := &eventbuscommon.AuthCredential{}

	switch {
	case eventBusAuth == nil || *eventBusAuth == v1alpha1.AuthStrategyNone:
		auth = &eventbuscommon.Auth{
			Strategy: v1alpha1.AuthStrategyNone,
		}
	case *eventBusAuth == v1alpha1.AuthStrategyJWT:
		// For JWT auth, we don't parse auth.yaml - just set the credential file path
		cred.CredentialFile = fmt.Sprintf("%s/credentials.creds", v1alpha1.EventBusAuthFileMountPath)
		auth = &eventbuscommon.Auth{
			Strategy:   v1alpha1.AuthStrategyJWT,
			Credential: cred,
		}
	default:
		// For Basic and Token auth, parse auth.yaml
		v := sharedutil.ViperWithLogging()
		v.SetConfigName("auth")
		v.SetConfigType("yaml")
		v.AddConfigPath(v1alpha1.EventBusAuthFileMountPath)
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
