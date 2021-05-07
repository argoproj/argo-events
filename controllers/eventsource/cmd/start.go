package cmd

import (
	"fmt"
	"os"
	"reflect"

	"go.uber.org/zap"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argoevents "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/controllers/eventsource"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	eventsourcev1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

const (
	eventSourceImageEnvVar = "EVENTSOURCE_IMAGE"
	sensorImageEnvVar      = "SENSOR_IMAGE"
)

func Start(namespaced bool, managedNamespace string) {
	logger := logging.NewArgoEventsLogger().Named(eventsource.ControllerName)
	eventSourceImage, defined := os.LookupEnv(eventSourceImageEnvVar)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", eventSourceImageEnvVar)
	}
	opts := ctrl.Options{
		MetricsBindAddress:     fmt.Sprintf(":%d", common.ControllerMetricsPort),
		HealthProbeBindAddress: ":8081",
	}
	if namespaced {
		opts.Namespace = managedNamespace
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		logger.Fatalw("unable to get a controller-runtime manager", zap.Error(err))
	}

	// Readyness probe
	if err := mgr.AddReadyzCheck("readiness", healthz.Ping); err != nil {
		logger.Fatalw("unable add a readiness check", zap.Error(err))
	}

	// Liveness probe
	if err := mgr.AddHealthzCheck("liveness", healthz.Ping); err != nil {
		logger.Fatalw("unable add a health check", zap.Error(err))
	}

	if err := eventsourcev1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		logger.Fatalw("unable to add EventSource scheme", zap.Error(err))
	}

	if err := eventbusv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		logger.Fatalw("unable to add EventBus scheme", zap.Error(err))
	}

	// A controller with DefaultControllerRateLimiter
	c, err := controller.New(eventsource.ControllerName, mgr, controller.Options{
		Reconciler: eventsource.NewReconciler(mgr.GetClient(), mgr.GetScheme(), eventSourceImage, logger),
	})
	if err != nil {
		logger.Fatalw("unable to set up individual controller", zap.Error(err))
	}

	// Watch EventSource and enqueue EventSource object key
	if err := c.Watch(&source.Kind{Type: &eventsourcev1alpha1.EventSource{}}, &handler.EnqueueRequestForObject{},
		predicate.Or(
			predicate.GenerationChangedPredicate{},
			// TODO: change to use LabelChangedPredicate with controller-runtime v0.8
			predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					if e.ObjectOld == nil {
						return false
					}
					if e.ObjectNew == nil {
						return false
					}
					return !reflect.DeepEqual(e.ObjectNew.GetLabels(), e.ObjectOld.GetLabels())
				}},
		)); err != nil {
		logger.Fatalw("unable to watch EventSources", zap.Error(err))
	}

	// Watch Deployments and enqueue owning EventSource key
	if err := c.Watch(&source.Kind{Type: &appv1.Deployment{}}, &handler.EnqueueRequestForOwner{OwnerType: &eventsourcev1alpha1.EventSource{}, IsController: true}, predicate.GenerationChangedPredicate{}); err != nil {
		logger.Fatalw("unable to watch Deployments", zap.Error(err))
	}

	// Watch Services and enqueue owning EventSource key
	if err := c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{OwnerType: &eventsourcev1alpha1.EventSource{}, IsController: true}, predicate.GenerationChangedPredicate{}); err != nil {
		logger.Fatalw("unable to watch Services", zap.Error(err))
	}

	logger.Infow("starting eventsource controller", "version", argoevents.GetVersion())
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Fatalw("unable to run eventsource controller", zap.Error(err))
	}
}
