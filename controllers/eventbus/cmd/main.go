package main

import (
	"flag"
	"fmt"
	"os"

	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/argoproj/argo-events/controllers/eventbus"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	natsStreamingEnvVar = "NATS_STREAMING_IMAGE"
)

var (
	log = ctrl.Log.WithName(eventbus.ControllerName)

	namespaced       bool
	managedNamespace string
)

func init() {
	flag.BoolVar(&namespaced, "namespaced", false, "run the controller as namespaced mode")
	flag.StringVar(&managedNamespace, "managed-namespace", os.Getenv("NAMESPACE"), "namespace that controller watches, default to the installation namespace")
	flag.Parse()
}

func main() {
	ecfg := uzap.NewProductionEncoderConfig()
	ecfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(ecfg)
	ctrl.SetLogger(zap.New(zap.UseDevMode(false), zap.WriteTo(os.Stdout), zap.Encoder(encoder)))
	mainLog := log.WithName("main")
	natsStreamingImage, defined := os.LookupEnv(natsStreamingEnvVar)
	if !defined {
		panic(fmt.Errorf("required environment variable '%s' not defined", natsStreamingEnvVar))
	}
	opts := ctrl.Options{}
	if namespaced {
		opts.Namespace = managedNamespace
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		panic(err)
	}
	err = v1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		mainLog.Error(err, "unable to add scheme")
		panic(err)
	}
	// A controller with DefaultControllerRateLimiter
	c, err := controller.New(eventbus.ControllerName, mgr, controller.Options{
		Reconciler: eventbus.NewReconciler(mgr.GetClient(), mgr.GetScheme(), natsStreamingImage, log.WithName("reconciler")),
	})
	if err != nil {
		mainLog.Error(err, "unable to set up individual controller")
		panic(err)
	}

	// Watch EventBus and enqueue EventBus object key
	if err := c.Watch(&source.Kind{Type: &v1alpha1.EventBus{}}, &handler.EnqueueRequestForObject{}); err != nil {
		mainLog.Error(err, "unable to watch EventBuses")
		panic(err)
	}

	// Watch ConfigMaps and enqueue owning EventBus key
	if err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventBus{}, IsController: true}); err != nil {
		mainLog.Error(err, "unable to watch ConfigMaps")
		panic(err)
	}

	// Watch Secrets and enqueue owning EventBus key
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventBus{}, IsController: true}); err != nil {
		mainLog.Error(err, "unable to watch Secrets")
		panic(err)
	}

	// Watch StatefulSets and enqueue owning EventBus key
	if err := c.Watch(&source.Kind{Type: &appv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventBus{}, IsController: true}); err != nil {
		mainLog.Error(err, "unable to watch StatefulSets")
		panic(err)
	}

	// Watch Services and enqueue owning EventBus key
	if err := c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventBus{}, IsController: true}); err != nil {
		mainLog.Error(err, "unable to watch Services")
		panic(err)
	}

	mainLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		mainLog.Error(err, "unable to run manager")
		panic(err)
	}
}
