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
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, sensorImage string, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, sensorImage: sensorImage, logger: logger}
}

func (r *reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	sensor := &v1alpha1.Sensor{}
	if err := r.client.Get(ctx, req.NamespacedName, sensor); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: sensor not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get sensor ctl", "request", req, "error", err)
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", sensor.Namespace).With("sensor", sensor.Name)
	sensorCopy := sensor.DeepCopy()
	reconcileErr := r.reconcile(ctx, sensorCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", "error", reconcileErr)
	}
	if r.needsUpdate(sensor, sensorCopy) {
		if err := r.client.Update(ctx, sensorCopy); err != nil {
			return reconcile.Result{}, err
		}
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, sensor *v1alpha1.Sensor) error {
	log := r.logger.With("namespace", sensor.Namespace).With("sensor", sensor.Name)
	if !sensor.DeletionTimestamp.IsZero() {
		log.Info("deleting sensor")
		// Finalizer logic should be added here.
		r.removeFinalizer(sensor)
		return nil
	}
	r.addFinalizer(sensor)

	sensor.Status.InitConditions()
	err := ValidateSensor(sensor)
	if err != nil {
		log.Error(err, "validation error")
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

func (r *reconciler) addFinalizer(s *v1alpha1.Sensor) {
	finalizers := sets.NewString(s.Finalizers...)
	finalizers.Insert(finalizerName)
	s.Finalizers = finalizers.List()
}

func (r *reconciler) removeFinalizer(s *v1alpha1.Sensor) {
	finalizers := sets.NewString(s.Finalizers...)
	finalizers.Delete(finalizerName)
	s.Finalizers = finalizers.List()
}

func (r *reconciler) needsUpdate(old, new *v1alpha1.Sensor) bool {
	if old == nil {
		return true
	}
	if !equality.Semantic.DeepEqual(old.Status, new.Status) {
		return true
	}
	if !equality.Semantic.DeepEqual(old.Finalizers, new.Finalizers) {
		return true
	}
	return false
}
