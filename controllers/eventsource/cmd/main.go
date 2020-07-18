package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
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

	"github.com/argoproj/argo-events/controllers/eventsource"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

const (
	eventSourceImageEnvVar = "EVENTSOURCE_IMAGE"
)

var (
	log = ctrl.Log.WithName(eventsource.ControllerName)

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
	ecfg := uzap.NewProductionEncoderConfig()
	ecfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(ecfg)
	ctrl.SetLogger(zap.New(zap.UseDevMode(false), zap.WriteTo(os.Stdout), zap.Encoder(encoder)))
	mainLog := log.WithName("main")
	eventSourceImage, defined := os.LookupEnv(eventSourceImageEnvVar)
	if !defined {
		panic(fmt.Errorf("required environment variable '%s' not defined", eventSourceImageEnvVar))
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
		mainLog.Error(err, "unable to add EventSource scheme")
		panic(err)
	}
	err = eventbusv1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		mainLog.Error(err, "unable to add EventBus scheme")
		panic(err)
	}
	// A controller with DefaultControllerRateLimiter
	c, err := controller.New(eventsource.ControllerName, mgr, controller.Options{
		Reconciler: eventsource.NewReconciler(mgr.GetClient(), mgr.GetScheme(), eventSourceImage, log.WithName("reconciler")),
	})
	if err != nil {
		mainLog.Error(err, "unable to set up individual controller")
		panic(err)
	}

	// Watch EventSource and enqueue EventSource object key
	if err := c.Watch(&source.Kind{Type: &v1alpha1.EventSource{}}, &handler.EnqueueRequestForObject{}); err != nil {
		mainLog.Error(err, "unable to watch EventSources")
		panic(err)
	}

	// Watch Deployments and enqueue owning EventSource key
	if err := c.Watch(&source.Kind{Type: &appv1.Deployment{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventSource{}, IsController: true}); err != nil {
		mainLog.Error(err, "unable to watch Deployments")
		panic(err)
	}

	// Watch Services and enqueue owning EventSource key
	if err := c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.EventSource{}, IsController: true}); err != nil {
		mainLog.Error(err, "unable to watch Services")
		panic(err)
	}

	mainLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		mainLog.Error(err, "unable to run manager")
		panic(err)
	}
}
