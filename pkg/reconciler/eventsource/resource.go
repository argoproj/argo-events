package eventsource

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	controllerscommon "github.com/argoproj/argo-events/pkg/reconciler/common"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
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
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AdaptorArgs are the args needed to create a sensor deployment
type AdaptorArgs struct {
	Image       string
	EventSource *aev1.EventSource
	Labels      map[string]string
}

// Reconcile does the real logic
func Reconcile(client controllerClient.Client, args *AdaptorArgs, logger *zap.SugaredLogger) error {
	ctx := context.Background()
	eventSource := args.EventSource
	eventBus := &aev1.EventBus{}
	eventBusName := v1alpha1.DefaultEventBusName
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
		patch := &appv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:          deploy.Name,
				Namespace:     deploy.Namespace,
				Labels:        expectedDeploy.Labels,
				ManagedFields: nil,
				Annotations: map[string]string{
					v1alpha1.AnnotationResourceSpecHash: expectedDeploy.Annotations[v1alpha1.AnnotationResourceSpecHash],
				},
			},
			Spec: expectedDeploy.Spec,
		}
		err = client.Patch(ctx, patch, controllerClient.Apply, controllerClient.ForceOwnership, controllerClient.FieldOwner("argo-events"))
		if err != nil {
			eventSource.Status.MarkDeployFailed("UpdateDeploymentFailed", "Failed to update existing deployment")
			logger.Errorw("error updating existing deployment", "error", err)
			return err
		}
		logger.Infow("deployment is updated", "deploymentName", deploy.Name)
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
		} else if existingSvc.Annotations != nil && existingSvc.Annotations[v1alpha1.AnnotationResourceSpecHash] != expectedSvc.Annotations[v1alpha1.AnnotationResourceSpecHash] {
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

func buildDeployment(args *AdaptorArgs, eventBus *aev1.EventBus) (*appv1.Deployment, error) {
	deploymentSpec, err := buildDeploymentSpec(args)
	if err != nil {
		return nil, err
	}
	eventSourceCopy := &aev1.EventSource{
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
	busConfigBytes, err := json.Marshal(eventBus.Status.Config)
	if err != nil {
		return nil, fmt.Errorf("failed marshal event bus config: %v", err)
	}

	env := []corev1.EnvVar{
		{
			Name:  v1alpha1.EnvVarEventSourceObject,
			Value: base64.StdEncoding.EncodeToString(eventSourceBytes),
		},
		{
			Name:  v1alpha1.EnvVarEventBusSubject,
			Value: fmt.Sprintf("eventbus-%s", args.EventSource.Namespace),
		},
		{
			Name:      v1alpha1.EnvVarPodName,
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:  v1alpha1.EnvVarLeaderElection,
			Value: args.EventSource.Annotations[v1alpha1.AnnotationLeaderElection],
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
		secretObjs = []interface{}{eventSourceCopy}
	case eventBus.Status.Config.JetStream != nil:
		tlsOptions := eventBus.Status.Config.JetStream.TLS
		if tlsOptions != nil {
			caCertSecret = tlsOptions.CACertSecret
			clientCertSecret = tlsOptions.ClientCertSecret
			clientKeySecret = tlsOptions.ClientKeySecret
		}
		accessSecret = eventBus.Status.Config.JetStream.AccessSecret
		secretObjs = []interface{}{eventSourceCopy}
	case eventBus.Status.Config.Kafka != nil:
		caCertSecret = nil
		clientCertSecret = nil
		clientKeySecret = nil
		accessSecret = nil
		secretObjs = []interface{}{eventSourceCopy, eventBus} // kafka requires secrets for sasl and tls
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
	volConfigMaps, volCofigMapMounts := sharedutil.VolumesFromSecretsOrConfigMaps(v1alpha1.ConfigMapKeySelectorType, eventSourceCopy)
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
			Namespace:    args.EventSource.Namespace,
			GenerateName: fmt.Sprintf("%s-eventsource-", args.EventSource.Name),
			Labels:       mergeLabels(args.EventSource.Labels, args.Labels),
		},
		Spec: *deploymentSpec,
	}
	if err := controllerscommon.SetObjectMeta(args.EventSource, deployment, aev1.EventSourceGroupVersionKind); err != nil {
		return nil, err
	}

	return deployment, nil
}

func buildDeploymentSpec(args *AdaptorArgs) (*appv1.DeploymentSpec, error) {
	eventSourceContainer := corev1.Container{
		Image:           args.Image,
		ImagePullPolicy: sharedutil.GetImagePullPolicy(),
		Args:            []string{"eventsource-service"},
		Ports: []corev1.ContainerPort{
			{Name: "metrics", ContainerPort: v1alpha1.EventSourceMetricsPort},
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
		},
		Spec: corev1.ServiceSpec{
			Ports:     ports,
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: eventSource.Spec.Service.ClusterIP,
			Selector:  args.Labels,
		},
	}

	labels := mergeLabels(args.EventSource.Labels, args.Labels)
	annotations := make(map[string]string)

	if args.EventSource.Spec.Service.Metadata != nil {
		if args.EventSource.Spec.Service.Metadata.Labels != nil {
			labels = mergeLabels(args.EventSource.Spec.Service.Metadata.Labels, labels)
		}
		if args.EventSource.Spec.Service.Metadata.Annotations != nil {
			annotations = mergeLabels(args.EventSource.Spec.Service.Metadata.Annotations, annotations)
		}
	}

	svc.ObjectMeta.SetLabels(labels)
	svc.ObjectMeta.SetAnnotations(annotations)

	if err := controllerscommon.SetObjectMeta(eventSource, svc, v1alpha1.EventSourceGroupVersionKind); err != nil {
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
