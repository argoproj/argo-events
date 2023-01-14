package installer

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// Installer is an interface for event bus installation
type Installer interface {
	Install(ctx context.Context) (*v1alpha1.BusConfig, error)
	// Uninsall only needs to handle those resources not cascade deleted.
	// For example, undeleted PVCs not automatically deleted when deleting a StatefulSet
	Uninstall(ctx context.Context) error
}

// Install function installs the event bus
func Install(ctx context.Context, eventBus *v1alpha1.EventBus, client client.Client, kubeClient kubernetes.Interface, config *controllers.GlobalConfig, logger *zap.SugaredLogger) error {
	installer, err := getInstaller(eventBus, client, kubeClient, config, logger)
	if err != nil {
		logger.Errorw("failed to an installer", zap.Error(err))
		return err
	}
	busConfig, err := installer.Install(ctx)
	if err != nil {
		logger.Errorw("installation error", zap.Error(err))
		return err
	}
	eventBus.Status.Config = *busConfig
	return nil
}

// GetInstaller returns Installer implementation
func getInstaller(eventBus *v1alpha1.EventBus, client client.Client, kubeClient kubernetes.Interface, config *controllers.GlobalConfig, logger *zap.SugaredLogger) (Installer, error) {
	if nats := eventBus.Spec.NATS; nats != nil {
		if nats.Exotic != nil {
			return NewExoticNATSInstaller(eventBus, logger), nil
		} else if nats.Native != nil {
			return NewNATSInstaller(client, eventBus, config, getLabels(eventBus), kubeClient, logger), nil
		}
	} else if js := eventBus.Spec.JetStream; js != nil {
		return NewJetStreamInstaller(client, eventBus, config, getLabels(eventBus), logger), nil
	}
	return nil, fmt.Errorf("invalid eventbus spec")
}

func getLabels(bus *v1alpha1.EventBus) map[string]string {
	return map[string]string{
		"controller":          "eventbus-controller",
		"eventbus-name":       bus.Name,
		common.LabelOwnerName: bus.Name,
	}
}

// Uninstall function will be run before the EventBus object is deleted,
// usually it could be used to uninstall the extra resources who would not be cleaned
// up when an EventBus is deleted. Most of the time this is not needed as all
// the dependency resources should have been deleted by owner references cascade
// deletion, but things like PVC created by StatefulSet need to be cleaned up
// separately.
//
// It could also be used to check if the EventBus object can be safely deleted.
func Uninstall(ctx context.Context, eventBus *v1alpha1.EventBus, client client.Client, kubeClient kubernetes.Interface, config *controllers.GlobalConfig, logger *zap.SugaredLogger) error {
	linkedEventSources, err := linkedEventSources(ctx, eventBus.Namespace, eventBus.Name, client)
	if err != nil {
		logger.Errorw("failed to query linked EventSources", zap.Error(err))
		return fmt.Errorf("failed to check if there is any EventSource linked, %w", err)
	}
	if linkedEventSources > 0 {
		return fmt.Errorf("Can not delete an EventBus with %v EventSources connected", linkedEventSources)
	}

	linkedSensors, err := linkedSensors(ctx, eventBus.Namespace, eventBus.Name, client)
	if err != nil {
		logger.Errorw("failed to query linked Sensors", zap.Error(err))
		return fmt.Errorf("failed to check if there is any Sensor linked, %w", err)
	}
	if linkedSensors > 0 {
		return fmt.Errorf("Can not delete an EventBus with %v Sensors connected", linkedSensors)
	}

	installer, err := getInstaller(eventBus, client, kubeClient, config, logger)
	if err != nil {
		logger.Errorw("failed to get an installer", zap.Error(err))
		return err
	}
	return installer.Uninstall(ctx)
}

func linkedEventSources(ctx context.Context, namespace, eventBusName string, c client.Client) (int, error) {
	esl := &eventsourcev1alpha1.EventSourceList{}
	if err := c.List(ctx, esl, &client.ListOptions{
		Namespace: namespace,
	}); err != nil {
		return 0, err
	}
	result := 0
	for _, es := range esl.Items {
		ebName := es.Spec.EventBusName
		if ebName == "" {
			ebName = common.DefaultEventBusName
		}
		if ebName == eventBusName {
			result++
		}
	}
	return result, nil
}

func linkedSensors(ctx context.Context, namespace, eventBusName string, c client.Client) (int, error) {
	sl := &sensorv1alpha1.SensorList{}
	if err := c.List(ctx, sl, &client.ListOptions{
		Namespace: namespace,
	}); err != nil {
		return 0, err
	}
	result := 0
	for _, s := range sl.Items {
		sName := s.Spec.EventBusName
		if sName == "" {
			sName = common.DefaultEventBusName
		}
		if sName == eventBusName {
			result++
		}
	}
	return result, nil
}
