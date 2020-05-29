package main

import (
	"fmt"
	"os"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/argoproj/argo-events/controllers/eventbus"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	natsImageEnvVar = "NATS_IMAGE"
)

var log = ctrl.Log.WithName("eventbus-controller")

func main() {
	ctrl.SetLogger(zap.Logger(false))
	mainLog := log.WithName("main")
	natsImage, defined := os.LookupEnv(natsImageEnvVar)
	if !defined {
		panic(fmt.Errorf("required environment variable '%s' not defined", natsImageEnvVar))
	}
	images := map[apicommon.EventBusType]string{apicommon.EventBusNATS: natsImage}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{})
	if err != nil {
		panic(err)
	}
	err = v1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		mainLog.Error(err, "unable to add scheme")
		panic(err)
	}
	c, err := controller.New(eventbus.ControllerName, mgr, controller.Options{
		Reconciler: eventbus.NewReconciler(mgr.GetClient(), mgr.GetScheme(), images, log.WithName("reconciler")),
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
