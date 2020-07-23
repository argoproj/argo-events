package eventbus

import (
	"context"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus/driver"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// GetDriver returns a Driver implementation
func GetDriver(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig, subject, clientID string) (driver.Driver, error) {
	logger := logging.FromContext(ctx).Desugar()
	var eventBusType apicommon.EventBusType
	var eventBusAuth *eventbusv1alpha1.AuthStrategy
	if eventBusConfig.NATS != nil {
		eventBusType = apicommon.EventBusNATS
		eventBusAuth = eventBusConfig.NATS.Auth
	} else {
		return nil, errors.New("invalid event bus")
	}
	var auth *driver.Auth
	cred := &driver.AuthCredential{}
	if eventBusAuth == nil || eventBusAuth == &eventbusv1alpha1.AuthStrategyNone {
		auth = &driver.Auth{
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
			logger.Error("failed to unmarshal auth.yaml", zap.Error(err))
			return nil, err
		}
		v.WatchConfig()
		v.OnConfigChange(func(e fsnotify.Event) {
			logger.Info("eventbus auth config file changed.")
			err = v.Unmarshal(cred)
			if err != nil {
				logger.Error("failed to unmarshal auth.yaml after reloading", zap.Error(err))
			}
		})
		auth = &driver.Auth{
			Strategy:    *eventBusAuth,
			Crendential: cred,
		}
	}

	var dvr driver.Driver
	switch eventBusType {
	case apicommon.EventBusNATS:
		dvr = driver.NewNATSStreaming(eventBusConfig.NATS.URL, *eventBusConfig.NATS.ClusterID, subject, clientID, auth, logger.Sugar())
	default:
		return nil, errors.New("invalid eventbus type")
	}
	return dvr, nil
}
