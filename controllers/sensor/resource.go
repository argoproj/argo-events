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
	err := client.Get(ctx, types.NamespacedName{Namespace: sensor.Namespace, Name: eventBusName}, eventBus)
	if err != nil {
		if apierrors.IsNotFound(err) {
			sensor.Status.MarkDeployFailed("EventBusNotFound", "EventBus not found.")
			logger.Error(err, "EventBus not found", "eventBusName", eventBusName)
			return errors.Errorf("eventbus %s not found", eventBusName)
		}
		sensor.Status.MarkDeployFailed("GetEventBusFailed", "Failed to get EventBus.")
		logger.Error(err, "failed to get EventBus", "eventBusName", eventBusName)
		return err
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
	sensorCopy := &v1alpha1.Sensor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: args.Sensor.Namespace,
			Name:      args.Sensor.Name,
		},
		Spec: args.Sensor.Spec,
	}
	sensorBytes, err := json.Marshal(sensorCopy)
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
		{
			Name:  common.EnvVarEventBusSubject,
			Value: fmt.Sprintf("eventbus-%s", args.Sensor.Namespace),
		},
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

func labelSelector(labelMap map[string]string) labels.Selector {
	return labels.SelectorFromSet(labelMap)
}
