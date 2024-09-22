package cmd

import (
	"crypto/tls"
	"os"
	"strconv"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsversiond "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
	"github.com/argoproj/argo-events/pkg/webhook"
)

const (
	serviceNameEnvVar     = "SERVICE_NAME"
	deploymentNameEnvVar  = "DEPLOYMENT_NAME"
	clusterRoleNameEnvVar = "CLUSTER_ROLE_NAME"
	namespaceEnvVar       = "NAMESPACE"
	portEnvVar            = "PORT"
)

func Start() {
	logger := logging.NewArgoEventsLogger().Named("webhook")
	kubeConfig, _ := os.LookupEnv(aev1.EnvVarKubeConfig)
	restConfig, err := sharedutil.GetClientConfig(kubeConfig)
	if err != nil {
		logger.Fatalw("failed to get kubeconfig", zap.Error(err))
	}
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	aeClient := eventsversiond.NewForConfigOrDie(restConfig).ArgoprojV1alpha1()

	namespace, defined := os.LookupEnv(namespaceEnvVar)
	if !defined {
		logger.Fatalf("required environment variable %q not defined", namespaceEnvVar)
	}

	portStr := sharedutil.LookupEnvStringOr(portEnvVar, "443")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Fatalf("port should be a number, environment variable %q not valid", portStr)
	}

	options := webhook.Options{
		ServiceName:     sharedutil.LookupEnvStringOr(serviceNameEnvVar, "events-webhook"),
		DeploymentName:  sharedutil.LookupEnvStringOr(deploymentNameEnvVar, "events-webhook"),
		ClusterRoleName: sharedutil.LookupEnvStringOr(clusterRoleNameEnvVar, "argo-events-webhook"),
		Namespace:       namespace,
		Port:            port,
		SecretName:      "events-webhook-certs",
		WebhookName:     "webhook.argo-events.argoproj.io",
		ClientAuth:      tls.VerifyClientCertIfGiven,
	}
	controller := webhook.AdmissionController{
		Client:           kubeClient,
		ArgoEventsClient: aeClient,
		Options:          options,
		Handlers: map[schema.GroupVersionKind]runtime.Object{
			aev1.EventBusGroupVersionKind:    &aev1.EventBus{},
			aev1.EventSourceGroupVersionKind: &aev1.EventSource{},
			aev1.SensorGroupVersionKind:      &aev1.Sensor{},
		},
		Logger: logger,
	}
	ctx := logging.WithLogger(signals.SetupSignalHandler(), logger)
	if err := controller.Run(ctx); err != nil {
		logger.Fatalw("Failed to create the admission controller", zap.Error(err))
	}
}
