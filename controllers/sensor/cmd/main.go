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
	"fmt"
	"os"

	"github.com/pkg/errors"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	appv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/argoproj/argo-events/controllers/sensor"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	sensorImageEnvVar = "SENSOR_IMAGE"
)

var (
	log = ctrl.Log.WithName(sensor.ControllerName)

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
	sensorImage, defined := os.LookupEnv(sensorImageEnvVar)
	if !defined {
		panic(fmt.Errorf("required environment variable '%s' not defined", sensorImageEnvVar))
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
		mainLog.Error(err, "unable to add Sensor scheme")
		panic(err)
	}
	err = eventbusv1alpha1.AddToScheme(mgr.GetScheme())
	if err != nil {
		mainLog.Error(err, "unable to add EventBus scheme")
		panic(err)
	}
	// A controller with DefaultControllerRateLimiter
	c, err := controller.New(sensor.ControllerName, mgr, controller.Options{
		Reconciler: sensor.NewReconciler(mgr.GetClient(), mgr.GetScheme(), namespace, sensorImage, log.WithName("reconciler")),
	})
	if err != nil {
		mainLog.Error(err, "unable to set up individual controller")
		panic(err)
	}

	// Watch Sensor and enqueue Sensor object key
	if err := c.Watch(&source.Kind{Type: &v1alpha1.Sensor{}}, &handler.EnqueueRequestForObject{}); err != nil {
		mainLog.Error(err, "unable to watch Sensors")
		panic(err)
	}

	// Watch Deployments and enqueue owning Sensor key
	if err := c.Watch(&source.Kind{Type: &appv1.Deployment{}}, &handler.EnqueueRequestForOwner{OwnerType: &v1alpha1.Sensor{}, IsController: true}); err != nil {
		mainLog.Error(err, "unable to watch Deployments")
		panic(err)
	}

	mainLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		mainLog.Error(err, "unable to run manager")
		panic(err)
	}
}
