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

package sensor

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/codefresh"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "sensor-controller"

	finalizerName = ControllerName
)

type reconciler struct {
	client client.Client
	scheme *runtime.Scheme

	sensorImage string
	logger      *zap.SugaredLogger

	cfAPI *codefresh.API
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, sensorImage string, logger *zap.SugaredLogger, cfAPI *codefresh.API) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, sensorImage: sensorImage, logger: logger, cfAPI: cfAPI}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sensor := &v1alpha1.Sensor{}
	if err := r.client.Get(ctx, req.NamespacedName, sensor); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: sensor not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get sensor ctl", zap.Any("request", req), zap.Error(err))
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", sensor.Namespace).With("sensor", sensor.Name)
	sensorCopy := sensor.DeepCopy()
	reconcileErr := r.reconcile(ctx, sensorCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
		r.cfAPI.ReportError(reconcileErr, codefresh.ErrorContext{
			ObjectMeta: sensor.ObjectMeta,
			TypeMeta: sensor.TypeMeta,
		})
	}
	if r.needsUpdate(sensor, sensorCopy) {
		if err := r.client.Update(ctx, sensorCopy); err != nil {
			return reconcile.Result{}, err
		}
	}
	if err := r.client.Status().Update(ctx, sensorCopy); err != nil {
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, sensor *v1alpha1.Sensor) error {
	log := r.logger.With("namespace", sensor.Namespace).With("sensor", sensor.Name)
	if !sensor.DeletionTimestamp.IsZero() {
		log.Info("deleting sensor")
		if controllerutil.ContainsFinalizer(sensor, finalizerName) {
			// Finalizer logic should be added here.
			controllerutil.RemoveFinalizer(sensor, finalizerName)
		}
		return nil
	}
	controllerutil.AddFinalizer(sensor, finalizerName)

	sensor.Status.InitConditions()
	if err := ValidateSensor(sensor); err != nil {
		log.Errorw("validation error", "error", err)
		return err
	}
	args := &AdaptorArgs{
		Image:  r.sensorImage,
		Sensor: sensor,
		Labels: map[string]string{
			"controller":           "sensor-controller",
			common.LabelSensorName: sensor.Name,
			common.LabelOwnerName:  sensor.Name,
		},
	}
	return Reconcile(r.client, args, log)
}

func (r *reconciler) needsUpdate(old, new *v1alpha1.Sensor) bool {
	if old == nil {
		return true
	}
	if !equality.Semantic.DeepEqual(old.Finalizers, new.Finalizers) {
		return true
	}
	return false
}
