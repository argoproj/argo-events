package eventsource

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
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// AdaptorArgs are the args needed to create a sensor deployment
type AdaptorArgs struct {
	Image       string
	EventSource *v1alpha1.EventSource
	Labels      map[string]string
}

// Reconcile does the real logic
func Reconcile(client client.Client, args *AdaptorArgs, logger *zap.SugaredLogger) error {
	ctx := context.Background()
	eventSource := args.EventSource
	eventBus := &eventbusv1alpha1.EventBus{}
	eventBusName := common.DefaultEventBusName
	if len(eventSource.Spec.EventBusName) > 0 {
		eventBusName = eventSource.Spec.EventBusName
	}
	err := client.Get(ctx, types.NamespacedName{Namespace: eventSource.Namespace, Name: eventBusName}, eventBus)
	if err != nil {
		if apierrors.IsNotFound(err) {
			eventSource.Status.MarkDeployFailed("EventBusNotFound", "EventBus not found.")
			logger.Errorw("EventBus not found", "eventBusName", eventBusName, "error", err)
			return fmt.Errorf("eventbus %s not found", eventBusName)
		}
		eventSource.Status.MarkDeployFailed("GetEventBusFailed", "Failed to get EventBus.")
		logger.Errorw("failed to get EventBus", "eventBusName", eventBusName, "error", err)
		return err
	}
	if !eventBus.Status.IsReady() {
		eventSource.Status.MarkDeployFailed("EventBusNotReady", "EventBus not ready.")
		logger.Errorw("event bus is not in ready status", "eventBusName", eventBusName, "error", err)
		return fmt.Errorf("eventbus not ready")
	}

	expectedDeploy, err := buildDeployment(args, eventBus)
	if err != nil {
		eventSource.Status.MarkDeployFailed("BuildDeploymentSpecFailed", "Failed to build Deployment spec.")
		logger.Errorw("failed to build deployment spec", "error", err)
		return err
	}

	deploy, err := getDeployment(ctx, client, args)
	if err != nil && !apierrors.IsNotFound(err) {
		eventSource.Status.MarkDeployFailed("GetDeploymentFailed", "Get existing deployment failed")
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
				eventSource.Status.MarkDeployFailed("UpdateDeploymentFailed", "Failed to update existing deployment")
				logger.Errorw("error updating existing deployment", "error", err)
				return err
			}
			logger.Infow("deployment is updated", "deploymentName", deploy.Name)
		}
	} else {
		err = client.Create(ctx, expectedDeploy)
		if err != nil {
			eventSource.Status.MarkDeployFailed("CreateDeploymentFailed", "Failed to create a deployment")
			logger.Errorw("error creating a deployment", "error", err)
			return err
		}
		logger.Infow("deployment is created", "deploymentName", expectedDeploy.Name)
	}
	// Service if any
	existingSvc, err := getService(ctx, client, args)
	if err != nil && !apierrors.IsNotFound(err) {
		eventSource.Status.MarkDeployFailed("GetServiceFailed", "Failed to get existing service")
		logger.Errorw("error getting existing service", "error", err)
		return err
	}
	expectedSvc, err := buildService(args)
	if err != nil {
		eventSource.Status.MarkDeployFailed("BuildServiceFailed", "Failed to build service spec")
		logger.Errorw("error building service spec", "error", err)
		return err
	}
	if expectedSvc == nil {
		if existingSvc != nil {
			err = client.Delete(ctx, existingSvc)
			if err != nil {
				eventSource.Status.MarkDeployFailed("DeleteServiceFailed", "Failed to delete existing service")
				logger.Errorw("error deleting existing service", "error", err)
				return err
			}
			logger.Infow("deleted existing service", "serviceName", existingSvc.Name)
		}
	} else {
		if existingSvc == nil {
			err = client.Create(ctx, expectedSvc)
			if err != nil {
				eventSource.Status.MarkDeployFailed("CreateServiceFailed", "Failed to create a service")
				logger.Errorw("error creating a service", "error", err)
				return err
			}
			logger.Infow("service is created", "serviceName", expectedSvc.Name)
		} else if existingSvc.Annotations != nil && existingSvc.Annotations[common.AnnotationResourceSpecHash] != expectedSvc.Annotations[common.AnnotationResourceSpecHash] {
			// To avoid service updating issues such as port name change, re-create it.
			err = client.Delete(ctx, existingSvc)
			if err != nil {
				eventSource.Status.MarkDeployFailed("DeleteServiceFailed", "Failed to delete existing service")
				logger.Errorw("error deleting existing service", "error", err)
				return err
			}
			err = client.Create(ctx, expectedSvc)
			if err != nil {
				eventSource.Status.MarkDeployFailed("RecreateServiceFailed", "Failed to re-create existing service")
				logger.Errorw("error re-creating existing service", "error", err)
				return err
			}
			logger.Infow("service is re-created", "serviceName", existingSvc.Name)
		}
	}
	eventSource.Status.MarkDeployed()
	return nil
}

func getDeployment(ctx context.Context, cl client.Client, args *AdaptorArgs) (*appv1.Deployment, error) {
	dl := &appv1.DeploymentList{}
	err := cl.List(ctx, dl, &client.ListOptions{
		Namespace:     args.EventSource.Namespace,
		LabelSelector: labelSelector(args.Labels),
	})
	if err != nil {
		return nil, err
	}
	for _, deploy := range dl.Items {
		if metav1.IsControlledBy(&deploy, args.EventSource) {
			return &deploy, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func buildDeployment(args *AdaptorArgs, eventBus *eventbusv1alpha1.EventBus) (*appv1.Deployment, error) {
	deploymentSpec, err := buildDeploymentSpec(args)
	if err != nil {
		return nil, err
	}
	eventSourceCopy := &v1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: args.EventSource.Namespace,
			Name:      args.EventSource.Name,
		},
		Spec: args.EventSource.Spec,
	}
	eventSourceBytes, err := json.Marshal(eventSourceCopy)
	if err != nil {
		return nil, fmt.Errorf("failed marshal eventsource spec")
	}
	encodedEventSourceSpec := base64.StdEncoding.EncodeToString(eventSourceBytes)
	envVars := []corev1.EnvVar{
		{
			Name:  common.EnvVarEventSourceObject,
			Value: encodedEventSourceSpec,
		},
		{
			Name:  common.EnvVarEventBusSubject,
			Value: fmt.Sprintf("eventbus-%s", args.EventSource.Namespace),
		},
		{
			Name:      common.EnvVarPodName,
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:  common.EnvVarLeaderElection,
			Value: args.EventSource.Annotations[common.AnnotationLeaderElection],
		},
	}

	busConfigBytes, err := json.Marshal(eventBus.Status.Config)
	if err != nil {
		return nil, fmt.Errorf("failed marshal event bus config: %v", err)
	}
	encodedBusConfig := base64.StdEncoding.EncodeToString(busConfigBytes)
	envVars = append(envVars, corev1.EnvVar{Name: common.EnvVarEventBusConfig, Value: encodedBusConfig})
	var accessSecret *corev1.SecretKeySelector
	switch {
	case eventBus.Status.Config.NATS != nil:
		natsConf := eventBus.Status.Config.NATS
		accessSecret = natsConf.AccessSecret
	case eventBus.Status.Config.JetStream != nil:
		jsConf := eventBus.Status.Config.JetStream
		accessSecret = jsConf.AccessSecret
	case eventBus.Status.Config.Kafka != nil:
		accessSecret = nil // todo
	default:
		return nil, fmt.Errorf("unsupported event bus")
	}

	volumes := deploymentSpec.Template.Spec.Volumes
	volumeMounts := deploymentSpec.Template.Spec.Containers[0].VolumeMounts
	emptyDirVolName := "tmp"
	volumes = append(volumes, corev1.Volume{
		Name: emptyDirVolName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: emptyDirVolName, MountPath: "/tmp"})

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
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "auth-volume", MountPath: common.EventBusAuthFileMountPath})
	}
	deploymentSpec.Template.Spec.Volumes = volumes
	deploymentSpec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

	envs := deploymentSpec.Template.Spec.Containers[0].Env
	envs = append(envs, envVars...)
	deploymentSpec.Template.Spec.Containers[0].Env = envs

	vols := []corev1.Volume{}
	volMounts := []corev1.VolumeMount{}
	oldVols := deploymentSpec.Template.Spec.Volumes
	oldVolMounts := deploymentSpec.Template.Spec.Containers[0].VolumeMounts
	if len(oldVols) > 0 {
		vols = append(vols, oldVols...)
	}
	if len(oldVolMounts) > 0 {
		volMounts = append(volMounts, oldVolMounts...)
	}
	volSecrets, volSecretMounts := common.VolumesFromSecretsOrConfigMaps(eventSourceCopy, common.SecretKeySelectorType)
	if len(volSecrets) > 0 {
		vols = append(vols, volSecrets...)
	}
	if len(volSecretMounts) > 0 {
		volMounts = append(volMounts, volSecretMounts...)
	}
	volConfigMaps, volCofigMapMounts := common.VolumesFromSecretsOrConfigMaps(eventSourceCopy, common.ConfigMapKeySelectorType)
	if len(volConfigMaps) > 0 {
		vols = append(vols, volConfigMaps...)
	}
	if len(volCofigMapMounts) > 0 {
		volMounts = append(volMounts, volCofigMapMounts...)
	}

	deploymentSpec.Template.Spec.Volumes = vols
	deploymentSpec.Template.Spec.Containers[0].VolumeMounts = volMounts

	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    args.EventSource.Namespace,
			GenerateName: fmt.Sprintf("%s-eventsource-", args.EventSource.Name),
			Labels:       mergeLabels(args.EventSource.Labels, args.Labels),
		},
		Spec: *deploymentSpec,
	}
	if err := controllerscommon.SetObjectMeta(args.EventSource, deployment, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return deployment, nil
}

func buildDeploymentSpec(args *AdaptorArgs) (*appv1.DeploymentSpec, error) {
	eventSourceContainer := corev1.Container{
		Image:           args.Image,
		ImagePullPolicy: common.GetImagePullPolicy(),
		Args:            []string{"eventsource-service"},
		Ports: []corev1.ContainerPort{
			{Name: "metrics", ContainerPort: common.EventSourceMetricsPort},
		},
	}
	if args.EventSource.Spec.Template != nil && args.EventSource.Spec.Template.Container != nil {
		if err := mergo.Merge(&eventSourceContainer, args.EventSource.Spec.Template.Container, mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	eventSourceContainer.Name = "main"
	podTemplateLabels := make(map[string]string)
	if args.EventSource.Spec.Template != nil && args.EventSource.Spec.Template.Metadata != nil &&
		len(args.EventSource.Spec.Template.Metadata.Labels) > 0 {
		for k, v := range args.EventSource.Spec.Template.Metadata.Labels {
			podTemplateLabels[k] = v
		}
	}
	for k, v := range args.Labels {
		podTemplateLabels[k] = v
	}

	replicas := args.EventSource.Spec.GetReplicas()
	spec := &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: args.Labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podTemplateLabels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					eventSourceContainer,
				},
			},
		},
	}
	if args.EventSource.Spec.Template != nil {
		if args.EventSource.Spec.Template.Metadata != nil {
			spec.Template.SetAnnotations(args.EventSource.Spec.Template.Metadata.Annotations)
		}
		spec.Template.Spec.ServiceAccountName = args.EventSource.Spec.Template.ServiceAccountName
		spec.Template.Spec.Volumes = args.EventSource.Spec.Template.Volumes
		spec.Template.Spec.SecurityContext = args.EventSource.Spec.Template.SecurityContext
		spec.Template.Spec.NodeSelector = args.EventSource.Spec.Template.NodeSelector
		spec.Template.Spec.Tolerations = args.EventSource.Spec.Template.Tolerations
		spec.Template.Spec.Affinity = args.EventSource.Spec.Template.Affinity
		spec.Template.Spec.ImagePullSecrets = args.EventSource.Spec.Template.ImagePullSecrets
		spec.Template.Spec.PriorityClassName = args.EventSource.Spec.Template.PriorityClassName
		spec.Template.Spec.Priority = args.EventSource.Spec.Template.Priority
	}
	return spec, nil
}

func getService(ctx context.Context, cl client.Client, args *AdaptorArgs) (*corev1.Service, error) {
	sl := &corev1.ServiceList{}
	err := cl.List(ctx, sl, &client.ListOptions{
		Namespace:     args.EventSource.Namespace,
		LabelSelector: labelSelector(args.Labels),
	})
	if err != nil {
		return nil, err
	}
	for _, svc := range sl.Items {
		if metav1.IsControlledBy(&svc, args.EventSource) {
			return &svc, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func buildService(args *AdaptorArgs) (*corev1.Service, error) {
	eventSource := args.EventSource
	if eventSource.Spec.Service == nil {
		return nil, nil
	}
	if len(eventSource.Spec.Service.Ports) == 0 {
		return nil, nil
	}
	// Use a ports copy otherwise it will update the oririnal Ports spec in EventSource
	ports := []corev1.ServicePort{}
	ports = append(ports, eventSource.Spec.Service.Ports...)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-eventsource-svc", eventSource.Name),
			Namespace: eventSource.Namespace,
			Labels:    mergeLabels(args.EventSource.Labels, args.Labels),
		},
		Spec: corev1.ServiceSpec{
			Ports:     ports,
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: eventSource.Spec.Service.ClusterIP,
			Selector:  args.Labels,
		},
	}
	if err := controllerscommon.SetObjectMeta(eventSource, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return svc, nil
}

func mergeLabels(eventBusLabels, given map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range eventBusLabels {
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
