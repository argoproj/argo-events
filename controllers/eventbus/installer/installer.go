package installer

import (
	"errors"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

// Installer is an interface for event bus installation
type Installer interface {
	Install() (*v1alpha1.BusConfig, error)
	// Uninsall only needs to handle those resources not cascade deleted.
	// For example, undeleted PVCs not automatically deleted when deleting a StatefulSet
	Uninstall() error
}

// Install function installs the event bus
func Install(eventBus *v1alpha1.EventBus, client client.Client, natsStreamingImage, natsMetricsImage string, logger *zap.SugaredLogger) error {
	installer, err := getInstaller(eventBus, client, natsStreamingImage, natsMetricsImage, logger)
	if err != nil {
		logger.Desugar().Error("failed to an installer", zap.Error(err))
		return err
	}
	busConfig, err := installer.Install()
	if err != nil {
		logger.Desugar().Error("installation error", zap.Error(err))
		return err
	}
	eventBus.Status.Config = *busConfig
	return nil
}

// GetInstaller returns Installer implementation
func getInstaller(eventBus *v1alpha1.EventBus, client client.Client, natsStreamingImage, natsMetricsImage string, logger *zap.SugaredLogger) (Installer, error) {
	if nats := eventBus.Spec.NATS; nats != nil {
		if nats.Exotic != nil {
			return NewExoticNATSInstaller(eventBus, logger), nil
		} else if nats.Native != nil {
			return NewNATSInstaller(client, eventBus, natsStreamingImage, natsMetricsImage, getLabels(eventBus), logger), nil
		}
	}
	return nil, errors.New("invalid eventbus spec")
}

func getLabels(bus *v1alpha1.EventBus) map[string]string {
	labels := bus.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[common.LabelControllerName] = "eventbus-controller"
	labels[common.LabelEventBusName] = bus.Name
	labels[common.LabelOwnerName] = bus.Name
	return labels
}

// Uninstall function uninstalls the extra resources who were not cleaned up
// when an eventbus was deleted. Most of the time this is not needed as all
// the dependency resources should have been deleted by owner references cascade
// deletion, but things like PVC created by StatefulSet need to be cleaned up
// separately.
func Uninstall(eventBus *v1alpha1.EventBus, client client.Client, natsStreamingImage, natsMetricsImage string, logger *zap.SugaredLogger) error {
	installer, err := getInstaller(eventBus, client, natsStreamingImage, natsMetricsImage, logger)
	if err != nil {
		logger.Desugar().Error("failed to get an installer", zap.Error(err))
		return err
	}
	return installer.Uninstall()
}
