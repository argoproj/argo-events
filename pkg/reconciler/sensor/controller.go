/*
Copyright 2020 The Argoproj Authors.

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
	"fmt"

	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "sensor-controller"

	finalizerName = ControllerName
)

type reconciler struct {
	client controllerClient.Client
	scheme *runtime.Scheme

	sensorImage string
	logger      *zap.SugaredLogger
}

// NewReconciler returns a new reconciler
func NewReconciler(client controllerClient.Client, scheme *runtime.Scheme, sensorImage string, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, sensorImage: sensorImage, logger: logger}
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
	ctx = logging.WithLogger(ctx, log)
	sensorCopy := sensor.DeepCopy()
	reconcileErr := r.reconcile(ctx, sensorCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
	}

	// Update the finalizers
	// We need to always do this to ensure that the field ownership is set correctly,
	// during the migration from client-side to server-side apply. Otherewise, users
	// may end up in a state where the finalizer cannot be removed automatically.
	patch := &v1alpha1.Sensor{
		Status: sensorCopy.Status,
		ObjectMeta: metav1.ObjectMeta{
			Name:          sensor.Name,
			Namespace:     sensor.Namespace,
			Finalizers:    sensorCopy.Finalizers,
			ManagedFields: nil,
		},
		TypeMeta: sensor.TypeMeta,
		Spec:     sensorCopy.Spec, // This is a bit inefficient, but we need to do it as the CRD is not configured properly to enable efficient merge
	}
	if len(patch.Finalizers) == 0 {
		patch.Finalizers = nil
	}
	if err := r.client.Patch(
		ctx,
		patch,
		controllerClient.Apply,
		controllerClient.ForceOwnership,
		controllerClient.FieldOwner("argo-events"),
	); err != nil {
		return reconcile.Result{}, err
	}

	// Update the status
	statusPatch := &v1alpha1.Sensor{
		Status: sensorCopy.Status,
		ObjectMeta: metav1.ObjectMeta{
			Name:          sensor.Name,
			Namespace:     sensor.Namespace,
			ManagedFields: nil,
		},
		TypeMeta: sensor.TypeMeta,
	}
	if err := r.client.Status().Patch(
		ctx,
		statusPatch,
		controllerClient.Apply,
		controllerClient.ForceOwnership,
		controllerClient.FieldOwner("argo-events"),
	); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, sensor *v1alpha1.Sensor) error {
	log := logging.FromContext(ctx)
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

	eventBus := &v1alpha1.EventBus{}
	eventBusName := v1alpha1.DefaultEventBusName
	if len(sensor.Spec.EventBusName) > 0 {
		eventBusName = sensor.Spec.EventBusName
	}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: sensor.Namespace, Name: eventBusName}, eventBus)
	if err != nil {
		if apierrors.IsNotFound(err) {
			sensor.Status.MarkDeployFailed("EventBusNotFound", "EventBus not found.")
			log.Errorw("EventBus not found", "eventBusName", eventBusName, "error", err)
			return fmt.Errorf("eventbus %s not found", eventBusName)
		}
		sensor.Status.MarkDeployFailed("GetEventBusFailed", "Failed to get EventBus.")
		log.Errorw("failed to get EventBus", "eventBusName", eventBusName, "error", err)
		return err
	}

	if err := ValidateSensor(sensor, eventBus); err != nil {
		log.Errorw("validation error", "error", err)
		return err
	}
	args := &AdaptorArgs{
		Image:  r.sensorImage,
		Sensor: sensor,
		Labels: map[string]string{
			"controller":             "sensor-controller",
			v1alpha1.LabelSensorName: sensor.Name,
			v1alpha1.LabelOwnerName:  sensor.Name,
		},
	}
	return Reconcile(r.client, eventBus, args, log)
}
