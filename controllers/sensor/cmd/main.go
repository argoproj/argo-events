/*
Copyright 2020 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"reflect"

	"github.com/pkg/errors"
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
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/controllers/sensor"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	sensorImageEnvVar = "SENSOR_IMAGE"
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
	logger := logging.NewArgoEventsLogger().Named(sensor.ControllerName)
	sensorImage, defined := os.LookupEnv(sensorImageEnvVar)
	if !defined {
		logger.Fatalf("required environment variable '%s' not defined", sensorImageEnvVar)
	}
	opts := ctrl.Options{
		MetricsBindAddress:     ":8080",
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
	err = mgr.AddReadyzCheck("readiness", healthz.Ping)
	if err != nil {
		logger.Fatalw("unable add a readiness check", zap.Error(err))
	}

	// Liveness probe
	err = mgr.AddHealthzCheck("liveness", healthz.Ping)
	if err != nil {
		logger.Fatalw("unable add a health check", zap.Error(err))
	}

	err = v1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		logger.Fatalw("unable to add Sensor scheme", zap.Error(err))
	}
	err = eventbusv1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		logger.Fatalw("uunable to add EventBus scheme", zap.Error(err))
	}
	// A controller with DefaultControllerRateLimiter
	c, err := controller.New(sensor.ControllerName, mgr, controller.Options{
		Reconciler: sensor.NewReconciler(mgr.GetClient(), mgr.GetScheme(), sensorImage, logger),
	})
	if err != nil {
		logger.Fatalw("unable to set up individual controller", zap.Error(err))
	}

	// Watch Sensor and enqueue Sensor object key
	if err := c.Watch(&source.Kind{Type: &v1alpha1.Sensor{}}, &handler.EnqueueRequestForObject{},
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
	if err := c.Watch(&source.Kind{Type: &appv1.Deployment{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.Sensor{}, IsController: true}, predicate.GenerationChangedPredicate{}); err != nil {
		logger.Fatalw("unable to watch Deployments", zap.Error(err))
	}

	logger.Infow("starting sensor controller", "version", argoevents.GetVersion())
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		logger.Fatalw("unable to run sensor controller", zap.Error(err))
	}
}
