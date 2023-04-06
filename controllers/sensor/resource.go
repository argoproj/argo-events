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

	"github.com/imdario/mergo"
	"go.uber.org/zap"
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
	Image  string
	Sensor *v1alpha1.Sensor
	Labels map[string]string
}

// Reconcile does the real logic
func Reconcile(client client.Client, eventBus *eventbusv1alpha1.EventBus, args *AdaptorArgs, logger *zap.SugaredLogger) error {
	ctx := context.Background()
	sensor := args.Sensor

	if eventBus == nil {
		sensor.Status.MarkDeployFailed("GetEventBusFailed", "Failed to get EventBus.")
		logger.Error("failed to get EventBus")
		return fmt.Errorf("failed to get EventBus")
	}

	eventBusName := common.DefaultEventBusName
	if len(sensor.Spec.EventBusName) > 0 {
		eventBusName = sensor.Spec.EventBusName
	}
	if !eventBus.Status.IsReady() {
		sensor.Status.MarkDeployFailed("EventBusNotReady", "EventBus not ready.")
		logger.Errorw("event bus is not in ready status", "eventBusName", eventBusName)
		return fmt.Errorf("eventbus not ready")
	}

	var configMap *corev1.ConfigMap
	if args.Sensor.Spec.LiveReload {
		liveReloadConfigMap, err := buildConfigMap(args)
		if err != nil {
			logger.Errorw("failed to build live reload config map", zap.Error(err))
			return err
		}

		if err := createOrUpdateConfigMap(ctx, client, liveReloadConfigMap); err != nil {
			logger.Errorw("failed to create/update live reload config map", zap.Error(err))
			return err
		}

		configMap = liveReloadConfigMap
	}

	expectedDeploy, err := buildDeployment(args, eventBus, configMap)
	if err != nil {
		sensor.Status.MarkDeployFailed("BuildDeploymentSpecFailed", "Failed to build Deployment spec.")
		logger.Errorw("failed to build deployment spec", "error", err)
		return err
	}
	deploy, err := getDeployment(ctx, client, args)
	if err != nil && !apierrors.IsNotFound(err) {
		sensor.Status.MarkDeployFailed("GetDeploymentFailed", "Get existing deployment failed")
		logger.Errorw("error getting existing deployment", "error", err)
		return err
	}
	if deploy != nil {
		if deploy.Annotations != nil && deploy.Annotations[common.AnnotationResourceSpecHash] != expectedDeploy.Annotations[common.AnnotationResourceSpecHash] {
			deploy.Spec = expectedDeploy.Spec
			deploy.SetLabels(expectedDeploy.Labels)
			deploy.Annotations[common.AnnotationResourceSpecHash] = expectedDeploy.Annotations[common.AnnotationResourceSpecHash]
			err = client.Update(ctx, deploy)
			if err != nil {
				sensor.Status.MarkDeployFailed("UpdateDeploymentFailed", "Failed to update existing deployment")
				logger.Errorw("error updating existing deployment", "error", err)
				return err
			}
			logger.Infow("deployment is updated", "deploymentName", deploy.Name)
		}
	} else {
		err = client.Create(ctx, expectedDeploy)
		if err != nil {
			sensor.Status.MarkDeployFailed("CreateDeploymentFailed", "Failed to create a deployment")
			logger.Errorw("error creating a deployment", "error", err)
			return err
		}
		logger.Infow("deployment is created", "deploymentName", expectedDeploy.Name)
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

func buildDeployment(args *AdaptorArgs, eventBus *eventbusv1alpha1.EventBus, configMap *corev1.ConfigMap) (*appv1.Deployment, error) {
	deploymentSpec, err := buildDeploymentSpec(args)
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
		return nil, fmt.Errorf("failed marshal sensor spec")
	}
	busConfigBytes, err := json.Marshal(eventBus.Status.Config)
	if err != nil {
		return nil, fmt.Errorf("failed marshal event bus config: %v", err)
	}

	env := []corev1.EnvVar{
		{
			Name:  common.EnvVarEventBusSubject,
			Value: fmt.Sprintf("eventbus-%s", args.Sensor.Namespace),
		},
		{
			Name:      common.EnvVarPodName,
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:  common.EnvVarLeaderElection,
			Value: args.Sensor.Annotations[common.AnnotationLeaderElection],
		},
		{
			Name:  common.EnvVarEventBusConfig,
			Value: base64.StdEncoding.EncodeToString(busConfigBytes),
		},
	}

	volumes := []corev1.Volume{
		{
			Name:         "tmp",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}

	if configMap != nil {
		const volumeName = "sensor-config-volume"
		env = append(env, corev1.EnvVar{
			Name:  common.EnvVarSensorFilePath,
			Value: fmt.Sprintf("%s/%s", common.SensorConfigMapMountPath, common.SensorConfigMapFilename),
		})
		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{
					Name: configMap.Name,
				}}},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: common.SensorConfigMapMountPath})
	} else {
		env = append(env, corev1.EnvVar{
			Name:  common.EnvVarSensorObject,
			Value: base64.StdEncoding.EncodeToString(sensorBytes),
		})
	}

	var secretObjs []interface{}
	var accessSecret *corev1.SecretKeySelector
	switch {
	case eventBus.Status.Config.NATS != nil:
		accessSecret = eventBus.Status.Config.NATS.AccessSecret
		secretObjs = []interface{}{sensorCopy}
	case eventBus.Status.Config.JetStream != nil:
		accessSecret = eventBus.Status.Config.JetStream.AccessSecret
		secretObjs = []interface{}{sensorCopy}
	case eventBus.Status.Config.Kafka != nil:
		accessSecret = nil
		secretObjs = []interface{}{sensorCopy, eventBus} // kafka requires secrets for sasl and tls
	default:
		return nil, fmt.Errorf("unsupported event bus")
	}

	if accessSecret != nil {
		// Mount the secret as volume instead of using envFrom to gain the ability
		// for the sensor deployment to auto reload when the secret changes
		volumes = append(volumes, corev1.Volume{
			Name: "auth-volume",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: accessSecret.Name,
					Items: []corev1.KeyToPath{
						{
							Key:  accessSecret.Key,
							Path: "auth.yaml",
						},
					},
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "auth-volume",
			MountPath: common.EventBusAuthFileMountPath,
		})
	}

	// secrets
	volSecrets, volSecretMounts := common.VolumesFromSecretsOrConfigMaps(common.SecretKeySelectorType, secretObjs...)
	volumes = append(volumes, volSecrets...)
	volumeMounts = append(volumeMounts, volSecretMounts...)

	// config maps
	volConfigMaps, volCofigMapMounts := common.VolumesFromSecretsOrConfigMaps(common.ConfigMapKeySelectorType, sensorCopy)
	volumeMounts = append(volumeMounts, volCofigMapMounts...)
	volumes = append(volumes, volConfigMaps...)

	deploymentSpec.Template.Spec.Containers[0].Env = append(deploymentSpec.Template.Spec.Containers[0].Env, env...)
	deploymentSpec.Template.Spec.Containers[0].VolumeMounts = append(deploymentSpec.Template.Spec.Containers[0].VolumeMounts, volumeMounts...)
	deploymentSpec.Template.Spec.Volumes = append(deploymentSpec.Template.Spec.Volumes, volumes...)

	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    args.Sensor.Namespace,
			GenerateName: fmt.Sprintf("%s-sensor-", args.Sensor.Name),
			Labels:       mergeLabels(args.Sensor.Labels, args.Labels),
		},
		Spec: *deploymentSpec,
	}
	if err := controllerscommon.SetObjectMeta(args.Sensor, deployment, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}

	return deployment, nil
}

func buildConfigMap(args *AdaptorArgs) (*corev1.ConfigMap, error) {
	serializedBytes, err := json.Marshal(args.Sensor)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("sensor-%s", args.Sensor.Name),
			Namespace: args.Sensor.Namespace,
		},
		Data: map[string]string{common.SensorConfigMapFilename: string(serializedBytes)},
	}, nil
}

func createOrUpdateConfigMap(ctx context.Context, client client.Client, configMap *corev1.ConfigMap) error {
	namespacedName := types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}
	err := client.Get(ctx, namespacedName, &corev1.ConfigMap{})
	if err != nil && apierrors.IsNotFound(err) {
		err = client.Create(ctx, configMap)
	} else if err == nil {
		err = client.Update(ctx, configMap)
	}
	return err
}

func buildDeploymentSpec(args *AdaptorArgs) (*appv1.DeploymentSpec, error) {
	replicas := args.Sensor.Spec.GetReplicas()
	sensorContainer := corev1.Container{
		Image:           args.Image,
		ImagePullPolicy: common.GetImagePullPolicy(),
		Args:            []string{"sensor-service"},
		Ports: []corev1.ContainerPort{
			{Name: "metrics", ContainerPort: common.SensorMetricsPort},
		},
	}
	if args.Sensor.Spec.Template != nil && args.Sensor.Spec.Template.Container != nil {
		if err := mergo.Merge(&sensorContainer, args.Sensor.Spec.Template.Container, mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	sensorContainer.Name = "main"
	podTemplateLabels := make(map[string]string)
	if args.Sensor.Spec.Template != nil && args.Sensor.Spec.Template.Metadata != nil &&
		len(args.Sensor.Spec.Template.Metadata.Labels) > 0 {
		for k, v := range args.Sensor.Spec.Template.Metadata.Labels {
			podTemplateLabels[k] = v
		}
	}
	for k, v := range args.Labels {
		podTemplateLabels[k] = v
	}
	spec := &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: args.Labels,
		},
		Replicas:             &replicas,
		RevisionHistoryLimit: args.Sensor.Spec.RevisionHistoryLimit,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podTemplateLabels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					sensorContainer,
				},
			},
		},
	}
	if args.Sensor.Spec.Template != nil {
		if args.Sensor.Spec.Template.Metadata != nil {
			spec.Template.SetAnnotations(args.Sensor.Spec.Template.Metadata.Annotations)
		}
		spec.Template.Spec.ServiceAccountName = args.Sensor.Spec.Template.ServiceAccountName
		spec.Template.Spec.Volumes = args.Sensor.Spec.Template.Volumes
		spec.Template.Spec.SecurityContext = args.Sensor.Spec.Template.SecurityContext
		spec.Template.Spec.NodeSelector = args.Sensor.Spec.Template.NodeSelector
		spec.Template.Spec.Tolerations = args.Sensor.Spec.Template.Tolerations
		spec.Template.Spec.Affinity = args.Sensor.Spec.Template.Affinity
		spec.Template.Spec.ImagePullSecrets = args.Sensor.Spec.Template.ImagePullSecrets
		spec.Template.Spec.PriorityClassName = args.Sensor.Spec.Template.PriorityClassName
		spec.Template.Spec.Priority = args.Sensor.Spec.Template.Priority
	}
	return spec, nil
}

func mergeLabels(sensorLabels, given map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range sensorLabels {
		result[k] = v
	}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func labelSelector(labelMap map[string]string) labels.Selector {
	return labels.SelectorFromSet(labelMap)
}
