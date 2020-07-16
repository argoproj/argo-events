package installer

import (
	"errors"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
func Install(eventBus *v1alpha1.EventBus, client client.Client, natsStreamingImage, natsMetricsImage string, logger logr.Logger) error {
	installer, err := getInstaller(eventBus, client, natsStreamingImage, natsMetricsImage, logger)
	if err != nil {
		logger.Error(err, "failed to an installer")
	}
	busConfig, err := installer.Install()
	if err != nil {
		logger.Error(err, "installation error")
		return err
	}
	eventBus.Status.Config = *busConfig
	return nil
}

// GetInstaller returns Installer implementation
func getInstaller(eventBus *v1alpha1.EventBus, client client.Client, natsStreamingImage, natsMetricsImage string, logger logr.Logger) (Installer, error) {
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
	return map[string]string{
		"controller":    "eventbus-controller",
		"eventbus-name": bus.Name,
	}
}

// Uninstall function uninstalls the extra resources who were not cleaned up
// when an eventbus was deleted. Most of the time this is not needed as all
// the dependency resources should have been deleted by owner references cascade
// deletion, but things like PVC created by StatefulSet need to be cleaned up
// separately.
func Uninstall(eventBus *v1alpha1.EventBus, client client.Client, natsStreamingImage, natsMetricsImage string, logger logr.Logger) error {
	installer, err := getInstaller(eventBus, client, natsStreamingImage, natsMetricsImage, logger)
	if err != nil {
		logger.Error(err, "failed to get an installer")
	}
	return installer.Uninstall()
}
