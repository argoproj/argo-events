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
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

func GetAuth(ctx context.Context, eventBusConfig eventbusv1alpha1.BusConfig) (*driver.Auth, error) {
	logger := logging.FromContext(ctx)

	var eventBusAuth *eventbusv1alpha1.AuthStrategy
	if eventBusConfig.NATS != nil {
		eventBusAuth = eventBusConfig.NATS.Auth
	} else if eventBusConfig.JetStream != nil {
		eventBusAuth = &eventbusv1alpha1.AuthStrategyToken
	} else {
		return nil, errors.New("invalid event bus")
	}
	var auth *driver.Auth
	cred := &driver.AuthCredential{}
	if eventBusAuth == nil || *eventBusAuth == eventbusv1alpha1.AuthStrategyNone {
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
		auth = &driver.Auth{
			Strategy:    *eventBusAuth,
			Crendential: cred,
		}

	}

	return auth, nil
}
