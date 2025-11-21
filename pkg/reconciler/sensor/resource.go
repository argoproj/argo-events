/*
Copyright 2018 The Argoproj Authors.

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
	"sort"

	"go.uber.org/zap"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	controllerscommon "github.com/argoproj/argo-events/pkg/reconciler/common"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// AdaptorArgs are the args needed to create a sensor deployment
type AdaptorArgs struct {
	Image  string
	Sensor *v1alpha1.Sensor
	Labels map[string]string
}

// Reconcile does the real logic
func Reconcile(client client.Client, eventBus *v1alpha1.EventBus, args *AdaptorArgs, logger *zap.SugaredLogger) error {
	ctx := context.Background()
	sensor := args.Sensor

	if eventBus == nil {
		sensor.Status.MarkDeployFailed("GetEventBusFailed", "Failed to get EventBus.")
		logger.Error("failed to get EventBus")
		return fmt.Errorf("failed to get EventBus")
	}

	eventBusName := v1alpha1.DefaultEventBusName
	if len(sensor.Spec.EventBusName) > 0 {
		eventBusName = sensor.Spec.EventBusName
	}
	if !eventBus.Status.IsReady() {
		sensor.Status.MarkDeployFailed("EventBusNotReady", "EventBus not ready.")
		logger.Errorw("event bus is not in ready status", "eventBusName", eventBusName)
		return fmt.Errorf("eventbus not ready")
	}

	expectedDeploy, err := buildDeployment(args, eventBus)
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
		if deploy.Annotations != nil && deploy.Annotations[v1alpha1.AnnotationResourceSpecHash] != expectedDeploy.Annotations[v1alpha1.AnnotationResourceSpecHash] {
			deploy.Spec = expectedDeploy.Spec
			deploy.SetLabels(expectedDeploy.Labels)
			deploy.Annotations[v1alpha1.AnnotationResourceSpecHash] = expectedDeploy.Annotations[v1alpha1.AnnotationResourceSpecHash]
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

func buildDeployment(args *AdaptorArgs, eventBus *v1alpha1.EventBus) (*appv1.Deployment, error) {
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
			Name:  v1alpha1.EnvVarSensorObject,
			Value: base64.StdEncoding.EncodeToString(sensorBytes),
		},
		{
			Name:  v1alpha1.EnvVarEventBusSubject,
			Value: fmt.Sprintf("eventbus-%s", args.Sensor.Namespace),
		},
		{
			Name:      v1alpha1.EnvVarPodName,
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:  v1alpha1.EnvVarLeaderElection,
			Value: args.Sensor.Annotations[v1alpha1.AnnotationLeaderElection],
		},
		{
			Name:  v1alpha1.EnvVarEventBusConfig,
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

	var secretObjs []interface{}
	var accessSecret *corev1.SecretKeySelector
	var caCertSecret *corev1.SecretKeySelector
	var clientCertSecret *corev1.SecretKeySelector
	var clientKeySecret *corev1.SecretKeySelector
	switch {
	case eventBus.Status.Config.NATS != nil:
		caCertSecret = nil
		clientCertSecret = nil
		clientKeySecret = nil
		accessSecret = eventBus.Status.Config.NATS.AccessSecret
		secretObjs = []interface{}{sensorCopy}
	case eventBus.Status.Config.JetStream != nil:
		tlsOptions := eventBus.Status.Config.JetStream.TLS
		if tlsOptions != nil {
			caCertSecret = tlsOptions.CACertSecret
			clientCertSecret = tlsOptions.ClientCertSecret
			clientKeySecret = tlsOptions.ClientKeySecret
		}
		accessSecret = eventBus.Status.Config.JetStream.AccessSecret
		secretObjs = []interface{}{sensorCopy}
	case eventBus.Status.Config.Kafka != nil:
		caCertSecret = nil
		clientCertSecret = nil
		clientKeySecret = nil
		accessSecret = nil
		secretObjs = []interface{}{sensorCopy, eventBus} // kafka requires secrets for sasl and tls
	default:
		return nil, fmt.Errorf("unsupported event bus")
	}

	if accessSecret != nil {
		// Mount the secret as volume instead of using envFrom to gain the ability
		// for the sensor deployment to auto reload when the secret changes

		// Determine auth strategy to decide how to mount the secret
		var authStrategy *v1alpha1.AuthStrategy
		if eventBus.Status.Config.JetStream != nil {
			authStrategy = eventBus.Status.Config.JetStream.Auth
		} else if eventBus.Status.Config.NATS != nil {
			authStrategy = eventBus.Status.Config.NATS.Auth
		}

		var items []corev1.KeyToPath
		if authStrategy != nil && *authStrategy == v1alpha1.AuthStrategyJWT {
			// For JWT auth, mount only the credentials file
			items = []corev1.KeyToPath{
				{
					Key:  accessSecret.Key,
					Path: "credentials.creds",
				},
			}
		} else {
			// For Basic/Token auth, mount as auth.yaml
			items = []corev1.KeyToPath{
				{
					Key:  accessSecret.Key,
					Path: "auth.yaml",
				},
			}
		}

		volumes = append(volumes, corev1.Volume{
			Name: "auth-volume",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: accessSecret.Name,
					Items:      items,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "auth-volume",
			MountPath: v1alpha1.EventBusAuthFileMountPath,
		})
	}

	uniqueCertVolumeMap := make(map[string][]corev1.KeyToPath)
	for _, secret := range []*corev1.SecretKeySelector{caCertSecret, clientCertSecret, clientKeySecret} {
		if secret != nil {
			uniqueCertVolumeMap[secret.Name] = append(uniqueCertVolumeMap[secret.Name], corev1.KeyToPath{
				Key:  secret.Key,
				Path: secret.Key,
			})
		}
	}

	// We deduplicate the certificate secret mounts to ensure every secret under the TLS config is only mounted once
	// because the secrets MUST be mounted at /argo-events/secrets/<secret-name>
	// in order for util.GetTLSConfig to work without modification
	for secretName, items := range uniqueCertVolumeMap {
		// The names of volumes MUST be valid DNS_LABELs; as the secret names are user-supplied,
		// we perform some input cleansing to ensure they conform
		volumeName := sharedutil.ConvertToDNSLabel(secretName)

		optional := false

		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
					Items:      items,
					Optional:   &optional,
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: fmt.Sprintf("/argo-events/secrets/%s", secretName),
		})
	}

	// secrets
	volSecrets, volSecretMounts := sharedutil.VolumesFromSecretsOrConfigMaps(v1alpha1.SecretKeySelectorType, secretObjs...)
	volumes = append(volumes, volSecrets...)
	volumeMounts = append(volumeMounts, volSecretMounts...)

	// config maps
	volConfigMaps, volCofigMapMounts := sharedutil.VolumesFromSecretsOrConfigMaps(v1alpha1.ConfigMapKeySelectorType, sensorCopy)
	volumeMounts = append(volumeMounts, volCofigMapMounts...)
	volumes = append(volumes, volConfigMaps...)

	// Order volumes and volumemounts based on name to make the order deterministic
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})
	sort.Slice(volumeMounts, func(i, j int) bool {
		return volumeMounts[i].Name < volumeMounts[j].Name
	})

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
	if err := controllerscommon.SetObjectMeta(args.Sensor, deployment, v1alpha1.SensorGroupVersionKind); err != nil {
		return nil, err
	}

	return deployment, nil
}

func buildDeploymentSpec(args *AdaptorArgs) (*appv1.DeploymentSpec, error) {
	replicas := args.Sensor.Spec.GetReplicas()
	sensorContainer := corev1.Container{
		Image:           args.Image,
		ImagePullPolicy: sharedutil.GetImagePullPolicy(),
		Args:            []string{"sensor-service"},
		Ports: []corev1.ContainerPort{
			{Name: "metrics", ContainerPort: v1alpha1.SensorMetricsPort},
		},
	}
	if x := args.Sensor.Spec.Template; x != nil && x.Container != nil {
		x.Container.ApplyToContainer(&sensorContainer)
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
