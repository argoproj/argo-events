package installer

import (
	"errors"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"go.uber.org/zap"
)

// exoticNATSInstaller is an inalleration implementation of exotic nats config.
type exoticNATSInstaller struct {
	eventBus *v1alpha1.EventBus

	logger *zap.SugaredLogger
}

// NewExoticNATSInstaller return a new exoticNATSInstaller
func NewExoticNATSInstaller(eventBus *v1alpha1.EventBus, logger *zap.SugaredLogger) Installer {
	return &exoticNATSInstaller{
		eventBus: eventBus,
		logger:   logger.Named("exotic-nats"),
	}
}

func (i *exoticNATSInstaller) Install() (*v1alpha1.BusConfig, error) {
	natsObj := i.eventBus.Spec.NATS
	if natsObj == nil || natsObj.Exotic == nil {
		return nil, errors.New("invalid request")
	}
	i.eventBus.Status.MarkDeployed("Skipped", "Skip deployment because of using exotic config.")
	i.eventBus.Status.MarkConfigured()
	i.logger.Info("use exotic config")
	busConfig := &v1alpha1.BusConfig{
		NATS: natsObj.Exotic,
	}
	return busConfig, nil
}

func (i *exoticNATSInstaller) Uninstall() error {
	i.logger.Info("nothing to uninstall")
	return nil
}
