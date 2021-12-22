package cmd

import (
	"crypto/tls"
	"os"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventbusv1alphal1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alphal1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	sensorv1alphal1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	eventbusclient "github.com/argoproj/argo-events/pkg/client/eventbus/clientset/versioned"
	eventsourceclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensorclient "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/argoproj/argo-events/webhook"
)

const (
	namespaceEnvVar = "NAMESPACE"
)

func Start(namespaced bool) {
	logger := logging.NewArgoEventsLogger().Named("webhook")
	kubeConfig, _ := os.LookupEnv(common.EnvVarKubeConfig)
	restConfig, err := common.GetClientConfig(kubeConfig)
	if err != nil {
		logger.Fatalw("failed to get kubeconfig", zap.Error(err))
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	eventBusClient := eventbusclient.NewForConfigOrDie(restConfig)
	eventSourceClient := eventsourceclient.NewForConfigOrDie(restConfig)
	sensorClient := sensorclient.NewForConfigOrDie(restConfig)

	namespace, defined := os.LookupEnv(namespaceEnvVar)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", namespaceEnvVar)
	}

	options := webhook.Options{
		ServiceName:    	"events-webhook",
		DeploymentName: 	"events-webhook",
		Namespace:      	namespace,
		Namespaced:     	namespaced,
		Port:           	443,
		SecretName:     	"events-webhook-certs",
		WebhookName:    	"webhook.argo-events.argoproj.io",
		ClientAuth:     	tls.VerifyClientCertIfGiven,
	}
	controller := webhook.AdmissionController{
		Client:            kubeClient,
		EventBusClient:    eventBusClient,
		EventSourceClient: eventSourceClient,
		SensorClient:      sensorClient,
		Options:           options,
		Handlers: map[schema.GroupVersionKind]runtime.Object{
			eventbusv1alphal1.SchemaGroupVersionKind:    &eventbusv1alphal1.EventBus{},
			eventsourcev1alphal1.SchemaGroupVersionKind: &eventsourcev1alphal1.EventSource{},
			sensorv1alphal1.SchemaGroupVersionKind:      &sensorv1alphal1.Sensor{},
		},
		Logger: logger,
	}
	ctx := logging.WithLogger(signals.SetupSignalHandler(), logger)
	if err := controller.Run(ctx); err != nil {
		logger.Fatalw("Failed to create the admission controller", zap.Error(err))
	}
}
