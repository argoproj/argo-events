package installer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// exoticJetStreamInstaller is an inalleration implementation of exotic jetstream config.
type exoticJetStreamInstaller struct {
	eventBus *v1alpha1.EventBus

	logger *zap.SugaredLogger
}

// NewExoticJetStreamInstaller return a new exoticJetStreamInstaller
func NewExoticJetStreamInstaller(eventBus *v1alpha1.EventBus, logger *zap.SugaredLogger) Installer {
	return &exoticJetStreamInstaller{
		eventBus: eventBus,
		logger:   logger.Named("exotic-jetstream"),
	}
}

func (i *exoticJetStreamInstaller) Install(ctx context.Context) (*v1alpha1.BusConfig, error) {
	JetStreamObj := i.eventBus.Spec.JetStreamExotic
	if JetStreamObj == nil {
		return nil, fmt.Errorf("invalid request")
	}
	i.eventBus.Status.MarkDeployed("Skipped", "Skip deployment because of using exotic config.")
	i.logger.Info("use exotic config")
	busConfig := &v1alpha1.BusConfig{
		JetStream: JetStreamObj,
	}
	return busConfig, nil
}

func (i *exoticJetStreamInstaller) Uninstall(ctx context.Context) error {
	i.logger.Info("nothing to uninstall")
	return nil
}
