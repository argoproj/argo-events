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
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/logging"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	// ControllerName is name of the controller
	ControllerName = "sensor-controller"

	CongigMapControllerName = "sensor-config-map-controller"
	finalizerName           = ControllerName
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
	if r.needsUpdate(sensor, sensorCopy) {
		// Use a DeepCopy to update, because it will be mutated afterwards, with empty Status.
		if err := r.client.Update(ctx, sensorCopy.DeepCopy()); err != nil {
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

	eventBus := &eventbusv1alpha1.EventBus{}
	eventBusName := common.DefaultEventBusName
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
			"controller":           "sensor-controller",
			common.LabelSensorName: sensor.Name,
			common.LabelOwnerName:  sensor.Name,
		},
	}
	return Reconcile(r.client, eventBus, args, log)
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

type congigmapReconciler struct {
	client client.Client
	scheme *runtime.Scheme

	logger *zap.SugaredLogger
}

func NewConfigMapReconciler(client client.Client, scheme *runtime.Scheme, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &congigmapReconciler{client: client, scheme: scheme, logger: logger}
}
func (r *congigmapReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.logger.With("namespace", request.Namespace).With("configmap", request.Name)
	log.Info(fmt.Sprintf("Reconciling ConfigMap: %s/%s", request.Namespace, request.Name))

	configMap := &corev1.ConfigMap{}
	configMapFound := true
	// Get the ConfigMap object
	if err := r.client.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			log.Warnw("WARNING: configmap not found, delete event", "request", request)
			configMapFound = false
		} else {
			log.Errorw("unable to get configmap", zap.Any("request", request), zap.Error(err))
			return ctrl.Result{}, err
		}
	}
	sensorName, hasSensorName := extractSensorName(request.Name)
	if !hasSensorName {
		log.Info("Unable to extract sensor name from configmap, configmap not associated with a sensor")
		return ctrl.Result{}, nil
	}

	sensor := &v1alpha1.Sensor{}
	// Get the sensor associated with configmap
	if err := r.client.Get(ctx, types.NamespacedName{Name: sensorName, Namespace: request.Namespace}, sensor); err != nil {
		if apierrors.IsNotFound(err) {
			log.Warnw("WARNING: associated sensor not found", "request", request)
			return reconcile.Result{}, nil
		}
		log.Errorw("unable to get sensor", zap.Any("request", request), zap.Error(err))
		return ctrl.Result{}, err
	}

	// Check if the ConfigMap has the expected owner reference
	if configMapFound && !metav1.IsControlledBy(configMap, sensor) {
		// ConfigMap does not have the expected owner reference, skip reconciliation
		log.Info(fmt.Sprintf("Skipping ConfigMap %s/%s, owner reference does not match", request.Namespace, request.Name))
		return reconcile.Result{}, nil
	}

	_, uuid, err := GetConfigMapDataAndUUID(sensor)
	if err != nil {
		log.Errorw("Failed to generated uuid from sensor", zap.Error(err))
		return reconcile.Result{}, err
	}
	// Case: configmap deleted, recreate it
	if !configMapFound || !configMap.DeletionTimestamp.IsZero() {
		// Only recreate the configmap if the uuid matches the request suffix
		if !strings.HasSuffix(request.Name, "-"+uuid) {
			log.Errorw("Configmap is obsolete and can be safely deleted", zap.Error(err))
			return reconcile.Result{}, nil
		}
		configMap.Name = request.Name
		configMap.Namespace = request.Namespace
		log.Info("recreating deleted configmap")
		err := r.client.Create(ctx, configMap)
		if err != nil {
			log.Errorw("Failed to recreate configmap", zap.Error(err))
			return reconcile.Result{}, err
		}
	}

	updatedConfigMap, err := getUpdatedConfigMap(configMap, sensor)
	if err != nil {
		log.Errorw("Error updating configmap", zap.Error(err))
		return reconcile.Result{}, err
	}
	if updatedConfigMap != nil {
		// Update the ConfigMap object
		if err := controllerscommon.SetObjectMeta(sensor, updatedConfigMap, v1alpha1.SchemaGroupVersionKind); err != nil {
			log.Errorw("Error setting configmap owner reference", zap.Error(err))
			return reconcile.Result{}, err
		}
		if err := r.client.Update(ctx, updatedConfigMap); err != nil {
			log.Errorw("Error updating configmap", zap.Error(err))
			return reconcile.Result{}, errors.Wrap(err, "failed to update ConfigMap")
		}

		log.Info(fmt.Sprintf("ConfigMap updated: %s/%s", request.Namespace, request.Name))
	}

	return reconcile.Result{}, nil
}

func extractSensorName(configMapName string) (string, bool) {
	configMapPrefix := "sensor-"
	if !strings.HasPrefix(configMapName, configMapPrefix) {
		return "", false
	}
	// volume name suffixed with -(md5)
	lastIndex := strings.LastIndex(configMapName, "-")
	return configMapName[len(configMapPrefix):lastIndex], true
}

func getUpdatedConfigMap(configMap *corev1.ConfigMap, sensor *v1alpha1.Sensor) (*corev1.ConfigMap, error) {
	serializedCRD, err := json.Marshal(sensor)
	if err != nil {
		return nil, err
	}
	serialziedCRDFromConfigMap, serialziedCRDFromConfigMapExists := configMap.Data[common.SensorConfigMapFilename]
	// Update cases: sensor file content has been modified or additional keys have been added
	if !serialziedCRDFromConfigMapExists || serialziedCRDFromConfigMap != string(serializedCRD) || len(configMap.Data) > 1 {
		configMap.Data = make(map[string]string)
		configMap.Data[common.SensorConfigMapFilename] = string(serializedCRD)
		return configMap, nil
	}
	return nil, nil
}
