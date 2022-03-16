package eventbus

import (
	"context"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common/logging"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	jetstreamsource "github.com/argoproj/argo-events/eventbus/jetstream/eventsource"
	jetstreamsensor "github.com/argoproj/argo-events/eventbus/jetstream/sensor"
	stansource "github.com/argoproj/argo-events/eventbus/stan/eventsource"
	stansensor "github.com/argoproj/argo-events/eventbus/stan/sensor"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func GetEventSourceDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, eventSourceName string, defaultSubject string) (eventbuscommon.EventSourceDriver, error) {
	auth, err := GetAuth(ctx, eventBusConfig)
	if err != nil {
		return nil, err
	}
	if eventSourceName == "" {
		return nil, errors.New("eventSourceName must be specified to create eventbus driver")
	}

	logger := logging.FromContext(ctx)

	logger.Infof("eventBusConfig: %+v", eventBusConfig)

	var eventBusType apicommon.EventBusType
	switch {
	case eventBusConfig.NATS != nil && eventBusConfig.JetStream != nil:
		return nil, errors.New("invalid event bus, NATS and Jetstream shouldn't both be specified")
	case eventBusConfig.NATS != nil:
		eventBusType = apicommon.EventBusNATS
	case eventBusConfig.JetStream != nil:
		eventBusType = apicommon.EventBusJetStream
	default:
		return nil, errors.New("invalid event bus")
	}

	var dvr eventbuscommon.EventSourceDriver
	switch eventBusType {
	case apicommon.EventBusNATS:
		if defaultSubject == "" {
			return nil, errors.New("subject must be specified to create NATS Streaming driver")
		}
		dvr = stansource.NewSourceNATSStreaming(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, eventSourceName, defaultSubject, auth, logger)
	case apicommon.EventBusJetStream:
		dvr, err = jetstreamsource.NewSourceJetstream(eventBusConfig.JetStream.URL, eventSourceName, auth, logger) // don't need to pass in subject because subjects will be derived from dependencies
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid eventbus type")
	}
	return dvr, nil
}

func GetSensorDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, sensorSpec *v1alpha1.Sensor) (eventbuscommon.SensorDriver, error) {
	auth, err := GetAuth(ctx, eventBusConfig)
	if err != nil {
		return nil, err
	}

	if sensorSpec == nil {
		return nil, errors.New("sensorSpec required for getting eventbus driver")
	}
	if sensorSpec.Name == "" {
		return nil, errors.New("sensorSpec name must be set for getting eventbus driver")
	}
	logger := logging.FromContext(ctx)

	var eventBusType apicommon.EventBusType
	switch {
	case eventBusConfig.NATS != nil && eventBusConfig.JetStream != nil:
		return nil, errors.New("invalid event bus, NATS and Jetstream shouldn't both be specified")
	case eventBusConfig.NATS != nil:
		eventBusType = apicommon.EventBusNATS
	case eventBusConfig.JetStream != nil:
		eventBusType = apicommon.EventBusJetStream
	default:
		return nil, errors.New("invalid event bus")
	}

	var dvr eventbuscommon.SensorDriver
	switch eventBusType {
	case apicommon.EventBusNATS:
		dvr = stansensor.NewSensorNATSStreaming(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, sensorSpec.Name, auth, logger)
	case apicommon.EventBusJetStream:
		dvr, err = jetstreamsensor.NewSensorJetstream(eventBusConfig.JetStream.URL, sensorSpec, auth, logger) // don't need to pass in subject because subjects will be derived from dependencies
	default:
		return nil, errors.New("invalid eventbus type")
	}
	return dvr, nil
}

func GetAuth(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig) (*Auth, error) {
	logger := logging.FromContext(ctx)

	var eventBusAuth *eventbusv1alpha1.AuthStrategy
	if eventBusConfig.NATS != nil {
		eventBusAuth = eventBusConfig.NATS.Auth
	} else if eventBusConfig.JetStream != nil {
		eventBusAuth = &eventbusv1alpha1.AuthStrategyToken
	} else {
		return nil, errors.New("invalid event bus")
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
			return nil, errors.Errorf("failed to load auth.yaml. err: %+v", err)
		}
		err = v.Unmarshal(cred)
		if err != nil {
			logger.Errorw("failed to unmarshal auth.yaml", zap.Error(err))
			return nil, err
		}
		v.WatchConfig()
		v.OnConfigChange(func(e fsnotify.Event) {
			logger.Info("eventbus auth config file changed.")
			err = v.Unmarshal(cred)
			if err != nil {
				logger.Errorw("failed to unmarshal auth.yaml after reloading", zap.Error(err))
			}
		})
		auth = &eventbuscommon.Auth{
			Strategy:    *eventBusAuth,
			Crendential: cred,
		}

	}

	return auth, nil
}
