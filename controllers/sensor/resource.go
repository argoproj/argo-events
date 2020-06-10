/*
Copyright 2018 BlackRock, Inc.

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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-logr/logr"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// AdaptorArgs are the args needed to create a sensor deployment
type AdaptorArgs struct {
	// controller namespace
	Namespace string
	Image     string
	Sensor    *v1alpha1.Sensor
	Labels    map[string]string
}

// Reconcile does the real logic
func Reconcile(client client.Client, args *AdaptorArgs, logger logr.Logger) error {
	ctx := context.Background()
	sensor := args.Sensor
	eventBus := &eventbusv1alpha1.EventBus{}
	eventBusName := "default"
	if len(sensor.Spec.EventBusName) > 0 {
		eventBusName = sensor.Spec.EventBusName
	}
	eventBusExisting := true
	err := client.Get(ctx, types.NamespacedName{Namespace: sensor.Namespace, Name: eventBusName}, eventBus)
	if err != nil {
		if apierrors.IsNotFound(err) {
			eventBusExisting = false
		} else {
			sensor.Status.MarkDeployFailed("GetEventBusFailed", "Failed to get EventBus.")
			logger.Error(err, "failed to get EventBus", "eventBusName", eventBusName)
			return err
		}
	}
	if !eventBusExisting {
		// DEPRECATED: Build lagecy resources
		// TODO: Remove following and error out.
		return reconcileLegacy(ctx, client, args, logger)
	}
	if !eventBus.Status.IsReady() {
		sensor.Status.MarkDeployFailed("EventBusNotReady", "EventBus not ready.")
		logger.Error(err, "event bus is not in ready status", "eventBusName", eventBusName)
		return errors.New("eventbus not ready")
	}
	expectedDeploy, err := buildDeployment(args, eventBus, logger)
	if err != nil {
		sensor.Status.MarkDeployFailed("BuildDeploymentSpecFailed", "Failed to build Deployment spec.")
		logger.Error(err, "failed to build deployment spec")
		return err
	}
	deploy, err := getDeployment(ctx, client, args)
	if err != nil && !apierrors.IsNotFound(err) {
		sensor.Status.MarkDeployFailed("GetDeploymentFailed", "Get existing deployment failed")
		logger.Error(err, "error getting existing deployment")
		return err
	}
	if deploy != nil {
		if deploy.Annotations != nil && deploy.Annotations[common.AnnotationResourceSpecHash] != expectedDeploy.Annotations[common.AnnotationResourceSpecHash] {
			deploy.Spec = expectedDeploy.Spec
			deploy.Annotations[common.AnnotationResourceSpecHash] = expectedDeploy.Annotations[common.AnnotationResourceSpecHash]
			err = client.Update(ctx, deploy)
			if err != nil {
				sensor.Status.MarkDeployFailed("UpdateDeploymentFailed", "Failed to update existing deployment")
				logger.Error(err, "error updating existing deployment")
				return err
			}
			logger.Info("deployment is updated", "deploymentName", deploy.Name)
		}
	} else {
		err = client.Create(ctx, expectedDeploy)
		if err != nil {
			sensor.Status.MarkDeployFailed("CreateDeploymentFailed", "Failed to create a deployment")
			logger.Error(err, "error creating a deployment")
			return err
		}
		logger.Info("deployment is created", "deploymentName", expectedDeploy.Name)
	}
	sensor.Status.MarkDeployed()
	return nil
}

// EventBus not existing create legacy service
// DEPRECATED
func reconcileLegacy(ctx context.Context, client client.Client, args *AdaptorArgs, logger logr.Logger) error {
	sensor := args.Sensor
	expectedDeploy, err := buildDeployment(args, nil, logger)
	if err != nil {
		sensor.Status.MarkDeployFailed("BuildDeploymentSpecFailed", "Failed to build Deployment spec.")
		logger.Error(err, "failed to build deployment spec")
		return err
	}
	expectedService, err := buildLegacyService(args, logger)
	if err != nil {
		sensor.Status.MarkDeployFailed("BuildServiceSpecFailed", "Failed to build legacy Service spec.")
		logger.Error(err, "failed to build service spec")
		return err
	}
	deploy, err := getDeployment(ctx, client, args)
	if err != nil && !apierrors.IsNotFound(err) {
		sensor.Status.MarkDeployFailed("GetDeploymentFailed", "Get existing deployment failed")
		logger.Error(err, "error getting existing deployment")
		return err
	}
	svc, err := getLegacyService(ctx, client, args)
	if err != nil && !apierrors.IsNotFound(err) {
		sensor.Status.MarkDeployFailed("GetLegacyServiceFailed", "Get existing legacy service failed")
		logger.Error(err, "error getting existing service")
		return err
	}

	if deploy != nil {
		if deploy.Annotations != nil && deploy.Annotations[common.AnnotationResourceSpecHash] != expectedDeploy.Annotations[common.AnnotationResourceSpecHash] {
			deploy.Spec = expectedDeploy.Spec
			deploy.Annotations[common.AnnotationResourceSpecHash] = expectedDeploy.Annotations[common.AnnotationResourceSpecHash]
			err = client.Update(ctx, deploy)
			if err != nil {
				sensor.Status.MarkDeployFailed("UpdateDeploymentFailed", "Failed to update existing deployment")
				logger.Error(err, "error updating existing deployment")
				return err
			}
			logger.Info("deployment is updated", "deploymentName", deploy.Name)
		}
	} else {
		err = client.Create(ctx, expectedDeploy)
		if err != nil {
			sensor.Status.MarkDeployFailed("CreateDeploymentFailed", "Failed to create a deployment")
			logger.Error(err, "error creating a deployment")
			return err
		}
		logger.Info("deployment is created", "deploymentName", expectedDeploy.Name)
	}

	if svc != nil {
		if svc.Annotations != nil && svc.Annotations[common.AnnotationResourceSpecHash] != expectedService.Annotations[common.AnnotationResourceSpecHash] {
			svc.Spec = expectedService.Spec
			svc.Annotations[common.AnnotationResourceSpecHash] = expectedService.Annotations[common.AnnotationResourceSpecHash]
			err = client.Update(ctx, svc)
			if err != nil {
				sensor.Status.MarkDeployFailed("UpdateServiceFailed", "Failed to update existing service")
				logger.Error(err, "error updating existing service")
				return err
			}
			logger.Info("service is updated", "serviceName", svc.Name)
		}
	} else {
		err = client.Create(ctx, expectedService)
		if err != nil {
			sensor.Status.MarkDeployFailed("CreateServiceFailed", "Failed to create a legacy service")
			logger.Error(err, "error creating a legacy service")
			return err
		}
		logger.Info("service is created", "serviceName", expectedService.Name)
	}
	sensor.Status.MarkDeployed()
	return nil
}

func getDeployment(ctx context.Context, cl client.Client, args *AdaptorArgs) (*appv1.Deployment, error) {
	dl := &appv1.DeploymentList{}
	err := cl.List(ctx, dl, &client.ListOptions{
		Namespace:     args.Sensor.Namespace,
		LabelSelector: labelSelector(args.Labels),
	})
	if err != nil {
		return nil, err
	}
	for _, deploy := range dl.Items {
		if metav1.IsControlledBy(&deploy, args.Sensor) {
			return &deploy, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func buildDeployment(args *AdaptorArgs, eventBus *eventbusv1alpha1.EventBus, log logr.Logger) (*appv1.Deployment, error) {
	deploymentSpec, err := buildDeploymentSpec(args, log)
	if err != nil {
		return nil, err
	}
	sensorBytes, err := json.Marshal(args.Sensor)
	if err != nil {
		return nil, errors.New("failed marshal sensor spec")
	}
	encodedSensorSpec := base64.StdEncoding.EncodeToString([]byte(sensorBytes))
	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarSensorObject,
			Value: encodedSensorSpec,
		},
		{
			Name:  common.SensorNamespace,
			Value: args.Sensor.Namespace,
		},
	}
	if eventBus != nil {
		if !eventBus.Status.IsReady() {
			return nil, errors.New("event bus is not ready")
		}
		busConfigBytes, err := json.Marshal(eventBus.Status.Config)
		if err != nil {
			return nil, errors.Errorf("failed marshal event bus config: %v", err)
		}
		encodedBusConfig := base64.StdEncoding.EncodeToString([]byte(busConfigBytes))
		envVars = append(envVars, corev1.EnvVar{Name: common.EnvVarEventBusConfig, Value: encodedBusConfig})
		if eventBus.Status.Config.NATS != nil {
			natsConf := eventBus.Status.Config.NATS
			if natsConf.Auth != nil && natsConf.AccessSecret != nil {
				// Mount the secret as volume instead of using evnFrom to gain the ability
				// for the sensor deployment to auto reload when the secret changes
				volumes := deploymentSpec.Template.Spec.Volumes
				volumes = append(volumes, corev1.Volume{
					Name: "auth-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: natsConf.AccessSecret.Name,
							Items: []corev1.KeyToPath{
								{
									Key:  natsConf.AccessSecret.Key,
									Path: "auth.yaml",
								},
							},
						},
					},
				})
				deploymentSpec.Template.Spec.Volumes = volumes
				volumeMounts := deploymentSpec.Template.Spec.Containers[0].VolumeMounts
				volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "auth-volume", MountPath: "/etc/eventbus/auth"})
				deploymentSpec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
			}
		} else {
			return nil, errors.New("unsupported event bus")
		}
	}
	envs := deploymentSpec.Template.Spec.Containers[0].Env
	envs = append(envs, envVars...)
	deploymentSpec.Template.Spec.Containers[0].Env = envs
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    args.Sensor.Namespace,
			GenerateName: fmt.Sprintf("%s-sensor-", args.Sensor.Name),
			Labels:       args.Labels,
		},
		Spec: *deploymentSpec,
	}
	if err := controllerscommon.SetObjectMeta(args.Sensor, deployment, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return deployment, nil
}

func buildDeploymentSpec(args *AdaptorArgs, log logr.Logger) (*appv1.DeploymentSpec, error) {
	// Deprecated spec, will be unsupported soon.
	if args.Sensor.Spec.Template.Spec != nil {
		log.Info("WARNING: spec.template.spec is DEPRECATED, it will be unsupported soon, please use spec.template.container")
		return buildLegacyDeploymentSpec(args.Sensor), nil
	}

	replicas := int32(1)
	sensorContainer := corev1.Container{
		Image:           args.Image,
		ImagePullPolicy: corev1.PullAlways,
	}
	if args.Sensor.Spec.Template.Container != nil {
		if err := mergo.Merge(&sensorContainer, args.Sensor.Spec.Template.Container, mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	sensorContainer.Name = "main"
	return &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: args.Labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: args.Labels,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: args.Sensor.Spec.Template.ServiceAccountName,
				Containers: []corev1.Container{
					sensorContainer,
				},
				Volumes:         args.Sensor.Spec.Template.Volumes,
				SecurityContext: args.Sensor.Spec.Template.SecurityContext,
			},
		},
	}, nil
}

// buildLegacyDeploymentSpec is deprecated, will be unsupported soon.
func buildLegacyDeploymentSpec(sensor *v1alpha1.Sensor) *appv1.DeploymentSpec {
	replicas := int32(1)
	labels := map[string]string{
		common.LabelObjectName: sensor.Name,
	}
	return &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: *sensor.Spec.Template.Spec,
		},
	}
}

// buildLegacyService builds a service that exposes sensor.
// DEPRECATED: This is to make it backward compatible.
func buildLegacyService(args *AdaptorArgs, log logr.Logger) (*corev1.Service, error) {
	port := common.SensorServerPort
	if args.Sensor.Spec.Subscription.HTTP != nil {
		port = int(args.Sensor.Spec.Subscription.HTTP.Port)
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-sensor", args.Sensor.Name),
			Labels:      args.Labels,
			Annotations: args.Sensor.Spec.ServiceAnnotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       intstr.FromInt(port).IntVal,
					TargetPort: intstr.FromInt(port),
				},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: args.Labels,
		},
	}
	if err := controllerscommon.SetObjectMeta(args.Sensor, service, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return service, nil
}

func getLegacyService(ctx context.Context, cl client.Client, args *AdaptorArgs) (*corev1.Service, error) {
	sl := &corev1.ServiceList{}
	err := cl.List(ctx, sl, &client.ListOptions{
		Namespace:     args.Sensor.Namespace,
		LabelSelector: labelSelector(args.Labels),
	})
	if err != nil {
		return nil, err
	}
	for _, svc := range sl.Items {
		if metav1.IsControlledBy(&svc, args.Sensor) {
			return &svc, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func labelSelector(labelMap map[string]string) labels.Selector {
	return labels.SelectorFromSet(labelMap)
}

// generate a random string as token with given length
func generateToken(length int) string {
	seeds := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = seeds[seededRand.Intn(len(seeds))]
	}
	return string(b)
}
