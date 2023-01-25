package installer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// exoticKafkaInstaller is an inalleration implementation of exotic kafka config.
type exoticKafkaInstaller struct {
	eventBus *v1alpha1.EventBus

	logger *zap.SugaredLogger
}

// NewExoticKafkaInstaller return a new exoticKafkaInstaller
func NewExoticKafkaInstaller(eventBus *v1alpha1.EventBus, logger *zap.SugaredLogger) Installer {
	return &exoticKafkaInstaller{
		eventBus: eventBus,
		logger:   logger.Named("exotic-kafka"),
	}
}

func (i *exoticKafkaInstaller) Install(ctx context.Context) (*v1alpha1.BusConfig, error) {
	kafkaObj := i.eventBus.Spec.Kafka
	if kafkaObj == nil || kafkaObj.Exotic == nil {
		return nil, fmt.Errorf("invalid request")
	}
	i.eventBus.Status.MarkDeployed("Skipped", "Skip deployment because of using exotic config.")
	i.logger.Info("use exotic config")
	busConfig := &v1alpha1.BusConfig{
		Kafka: kafkaObj.Exotic,
	}
	return busConfig, nil
}

func (i *exoticKafkaInstaller) Uninstall(ctx context.Context) error {
	i.logger.Info("nothing to uninstall")
	return nil
}
