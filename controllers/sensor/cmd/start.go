package cmd

import (
	"fmt"
	"os"
	"reflect"

	"go.uber.org/zap"
	appv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argoevents "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/codefresh"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/controllers/sensor"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	sensorImageEnvVar = "SENSOR_IMAGE"
)

func Start(namespaced bool, managedNamespace string) {
	ctx := signals.SetupSignalHandler()
	logger := logging.NewArgoEventsLogger().Named(sensor.ControllerName)
	sensorImage, defined := os.LookupEnv(sensorImageEnvVar)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", sensorImageEnvVar)
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

	if err := sensorv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		logger.Fatalw("unable to add Sensor scheme", zap.Error(err))
	}

	if err := eventbusv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		logger.Fatalw("uunable to add EventBus scheme", zap.Error(err))
	}

	cfAPI, err := codefresh.NewAPI(logging.WithLogger(ctx, logger), managedNamespace)
	if err != nil {
		logger.Warnw("WARNING: unable to initialise Codefresh API", zap.Error(err))
	}

	// A controller with DefaultControllerRateLimiter
	c, err := controller.New(sensor.ControllerName, mgr, controller.Options{
		Reconciler: sensor.NewReconciler(mgr.GetClient(), mgr.GetScheme(), sensorImage, logger, cfAPI),
	})
	if err != nil {
		logger.Fatalw("unable to set up individual controller", zap.Error(err))
	}

	// Watch Sensor and enqueue Sensor object key
	if err := c.Watch(&source.Kind{Type: &sensorv1alpha1.Sensor{}}, &handler.EnqueueRequestForObject{},
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
		logger.Fatalw("unable to watch Sensors", zap.Error(err))
	}

	// Watch Deployments and enqueue owning Sensor key
	if err := c.Watch(&source.Kind{Type: &appv1.Deployment{}}, &handler.EnqueueRequestForOwner{OwnerType: &sensorv1alpha1.Sensor{}, IsController: true}, predicate.GenerationChangedPredicate{}); err != nil {
		logger.Fatalw("unable to watch Deployments", zap.Error(err))
	}

	logger.Infow("starting sensor controller", "version", argoevents.GetVersion())
	if err := mgr.Start(ctx); err != nil {
		logger.Fatalw("unable to run sensor controller", zap.Error(err))
	}
}
