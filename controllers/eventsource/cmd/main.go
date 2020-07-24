package main

import (
	"flag"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argoevents "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/controllers/eventsource"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

const (
	eventSourceImageEnvVar = "EVENTSOURCE_IMAGE"
)

var (
	namespace        string
	namespaced       bool
	managedNamespace string
)

func init() {
	ns, defined := os.LookupEnv("NAMESPACE")
	if !defined {
		panic(errors.New("required environment variable NAMESPACE not defined"))
	}
	namespace = ns

	flag.BoolVar(&namespaced, "namespaced", false, "run the controller as namespaced mode")
	flag.StringVar(&managedNamespace, "managed-namespace", namespace, "namespace that controller watches, default to the installation namespace")
	flag.Parse()
}

func main() {
	logger := logging.NewArgoEventsLogger().Named(eventsource.ControllerName)
	eventSourceImage, defined := os.LookupEnv(eventSourceImageEnvVar)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", eventSourceImageEnvVar)
	}
	opts := ctrl.Options{}
	if namespaced {
		opts.Namespace = managedNamespace
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		logger.Desugar().Fatal("unable to get a controller-runtime manager", zap.Error(err))
	}
	err = v1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		logger.Desugar().Fatal("unable to add EventSource scheme", zap.Error(err))
	}
	err = eventbusv1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		logger.Desugar().Fatal("unable to add EventBus scheme", zap.Error(err))
	}
	// A controller with DefaultControllerRateLimiter
	c, err := controller.New(eventsource.ControllerName, mgr, controller.Options{
		Reconciler: eventsource.NewReconciler(mgr.GetClient(), mgr.GetScheme(), eventSourceImage, logger),
	})
	if err != nil {
		logger.Desugar().Fatal("unable to set up individual controller", zap.Error(err))
	}

	// Watch EventSource and enqueue EventSource object key
	if err := c.Watch(&source.Kind{Type: &v1alpha1.EventSource{}}, &handler.EnqueueRequestForObject{}); err != nil {
		logger.Desugar().Fatal("unable to watch EventSources", zap.Error(err))
	}

	// Watch Deployments and enqueue owning EventSource key
	if err := c.Watch(&source.Kind{Type: &appv1.Deployment{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventSource{}, IsController: true}); err != nil {
		logger.Desugar().Fatal("unable to watch Deployments", zap.Error(err))
	}

	// Watch Services and enqueue owning EventSource key
	if err := c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventSource{}, IsController: true}); err != nil {
		logger.Desugar().Fatal("unable to watch Services", zap.Error(err))
	}

	logger.Infow("starting eventsource controller", "version", argoevents.GetVersion())
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Desugar().Fatal("unable to run eventsource controller", zap.Error(err))
	}
}
