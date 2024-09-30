package eventsource

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
	"github.com/imdario/mergo"
)

const (
	finalizerName = v1alpha1.ControllerEventSource
)

type reconciler struct {
	client client.Client
	scheme *runtime.Scheme

	eventSourceImage string
	logger           *zap.SugaredLogger
}

// NewReconciler returns a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, eventSourceImage string, logger *zap.SugaredLogger) reconcile.Reconciler {
	return &reconciler{client: client, scheme: scheme, eventSourceImage: eventSourceImage, logger: logger}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	eventSource := &v1alpha1.EventSource{}
	if err := r.client.Get(ctx, req.NamespacedName, eventSource); err != nil {
		if apierrors.IsNotFound(err) {
			r.logger.Warnw("WARNING: eventsource not found", "request", req)
			return reconcile.Result{}, nil
		}
		r.logger.Errorw("unable to get eventsource", "request", req, zap.Error(err))
		return ctrl.Result{}, err
	}
	log := r.logger.With("namespace", eventSource.Namespace).With("eventSource", eventSource.Name)
	ctx = logging.WithLogger(ctx, log)
	esCopy := eventSource.DeepCopy()
	reconcileErr := r.reconcile(ctx, esCopy)
	if reconcileErr != nil {
		log.Errorw("reconcile error", zap.Error(reconcileErr))
	}
	esCopy.Status.LastUpdated = metav1.Now()
	if !equality.Semantic.DeepEqual(eventSource.Finalizers, esCopy.Finalizers) {
		patchYaml := "metadata:\n  finalizers: [" + strings.Join(esCopy.Finalizers, ",") + "]"
		patchJson, _ := yaml.YAMLToJSON([]byte(patchYaml))
		if err := r.client.Patch(ctx, eventSource, client.RawPatch(types.MergePatchType, []byte(patchJson))); err != nil {
			return ctrl.Result{}, err
		}
	}
	if err := r.client.Status().Update(ctx, esCopy); err != nil {
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile does the real logic
func (r *reconciler) reconcile(ctx context.Context, eventSource *v1alpha1.EventSource) error {
	log := logging.FromContext(ctx)
	if !eventSource.DeletionTimestamp.IsZero() {
		log.Info("Deleting eventsource")
		if controllerutil.ContainsFinalizer(eventSource, finalizerName) {
			// We don't add finalizer anymore, keep the removing logic for backward compatibility
			controllerutil.RemoveFinalizer(eventSource, finalizerName)
		}
		return nil
	}

	eventSource.Status.InitConditions()
	eventSource.Status.SetObservedGeneration(eventSource.Generation)
	if err := ValidateEventSource(eventSource); err != nil {
		log.Errorw("Validation error", zap.Error(err))
		eventSource.Status.MarkSourcesNotProvided("InvalidEventSource", err.Error())
		return err
	}

	if err := r.orchestrateResources(ctx, eventSource); err != nil {
		log.Errorw("Error orchestrating resources", zap.Error(err))
		eventSource.Status.MarkDeployFailed("OrchestrateResourcesFailed", err.Error())
		return err
	}
	eventSource.Status.MarkDeployed()
	return nil
}

func (r *reconciler) orchestrateResources(ctx context.Context, eventSource *v1alpha1.EventSource) error {
	log := logging.FromContext(ctx)
	eventBus := &v1alpha1.EventBus{}
	eventBusName := v1alpha1.DefaultEventBusName
	if len(eventSource.Spec.EventBusName) > 0 {
		eventBusName = eventSource.Spec.EventBusName
	}
	err := r.client.Get(ctx, types.NamespacedName{Namespace: eventSource.Namespace, Name: eventBusName}, eventBus)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// eventSource.Status.MarkDeployFailed("EventBusNotFound", "EventBus not found.")
			log.Errorw("EventBus not found", "eventBusName", eventBusName, zap.Error(err))
			return fmt.Errorf("eventbus %s not found", eventBusName)
		}
		return fmt.Errorf("failed to get eventbus %q, %w", eventBusName, err)
	}
	if !eventBus.Status.IsReady() {
		// eventSource.Status.MarkDeployFailed("EventBusNotReady", "EventBus not ready.")
		return fmt.Errorf("eventbus %q not ready", eventBusName)
	}

	if err := r.createOrUpdateDeployment(ctx, eventBus, eventSource); err != nil {
		return fmt.Errorf("failed to create/update a deployment, %w", err)
	}

	// Create service if any
	if eventSource.Spec.Service != nil {
		if err := r.createOrUpdateService(ctx, eventSource); err != nil {
			return fmt.Errorf("failed to create/update a service, %w", err)
		}
	}
	return nil
}

func (r *reconciler) createOrUpdateDeployment(ctx context.Context, eventBus *v1alpha1.EventBus, eventSource *v1alpha1.EventSource) error {
	log := logging.FromContext(ctx)

	// Clean up legacy deployment
	// TODO: remove this in a future rlease
	legacyDeploy, err := r.findLegacyDeployment(ctx, eventSource)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Errorw("error finding legacy deployment", zap.Error(err))
			return err
		}
	} else {
		_ = r.client.Delete(ctx, legacyDeploy)
	}

	expectedDeploy, err := r.buildDeployment(eventBus, eventSource)
	if err != nil {
		return fmt.Errorf("failed to build deployment spec, %w", err)
	}

	deployHash := expectedDeploy.GetAnnotations()[v1alpha1.KeyHash]

	existingDeploy := &appv1.Deployment{}
	needToCreate := false
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: eventSource.Namespace, Name: expectedDeploy.Name}, existingDeploy); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Errorw("Failed to find existing deployment", zap.String("deployment", expectedDeploy.Name), zap.Error(err))
			return fmt.Errorf("failed to find existing deployment, %w", err)
		} else {
			needToCreate = true
		}
	} else {
		if existingDeploy.GetAnnotations()[v1alpha1.KeyHash] != deployHash {
			// Delete and recreate, to avoid updating immutable fields problem.
			if err := r.client.Delete(ctx, expectedDeploy); err != nil {
				log.Errorw("Failed to delete the outdated daemon deployment", zap.String("deployment", existingDeploy.Name), zap.Error(err))
				return fmt.Errorf("failed to delete an outdated deployment, %w", err)
			}
			needToCreate = true
		}
	}
	if needToCreate {
		if err := r.client.Create(ctx, existingDeploy); err != nil && !apierrors.IsAlreadyExists(err) {
			log.Errorw("Failed to create a deployment", zap.String("deployment", expectedDeploy.Name), zap.Error(err))
			return fmt.Errorf("failed to create a deployment, %w", err)
		}
		log.Infow("Succeeded to create/recreate a deployment", zap.String("deployment", expectedDeploy.Name))
	}
	return nil
}

func (r *reconciler) buildDeployment(eventBus *v1alpha1.EventBus, eventSource *v1alpha1.EventSource) (*appv1.Deployment, error) {
	labels := map[string]string{
		v1alpha1.KeyManagedBy:       v1alpha1.ControllerEventSource,
		v1alpha1.KeyComponent:       v1alpha1.ComponentEventSource,
		v1alpha1.KeyPartOf:          v1alpha1.Project,
		v1alpha1.KeyAppName:         eventSource.GetDeploymentName(),
		v1alpha1.KeyEventSourceName: eventSource.Name,
	}
	deploymentSpec, err := r.buildDeploymentSpec(eventSource, labels)
	if err != nil {
		return nil, err
	}
	eventSourceCopy := &v1alpha1.EventSource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: eventSource.Namespace,
			Name:      eventSource.Name,
		},
		Spec: eventSource.Spec,
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
			Value: fmt.Sprintf("eventbus-%s", eventSource.Namespace),
		},
		{
			Name:      v1alpha1.EnvVarPodName,
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
		{
			Name:  v1alpha1.EnvVarLeaderElection,
			Value: eventSource.Annotations[v1alpha1.AnnotationLeaderElection],
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
	switch {
	case eventBus.Status.Config.NATS != nil:
		accessSecret = eventBus.Status.Config.NATS.AccessSecret
		secretObjs = []interface{}{eventSourceCopy}
	case eventBus.Status.Config.JetStream != nil:
		accessSecret = eventBus.Status.Config.JetStream.AccessSecret
		secretObjs = []interface{}{eventSourceCopy}
	case eventBus.Status.Config.Kafka != nil:
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

	hash := sharedutil.MustHash(deploymentSpec)

	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: eventSource.Namespace,
			Name:      eventSource.GetDeploymentName(),
			Labels:    labels,
			Annotations: map[string]string{
				v1alpha1.KeyHash: hash,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(eventSource.GetObjectMeta(), v1alpha1.EventBusGroupVersionKind),
			},
		},
		Spec: *deploymentSpec,
	}
	return deployment, nil
}

func (r *reconciler) buildDeploymentSpec(eventSource *v1alpha1.EventSource, labels map[string]string) (*appv1.DeploymentSpec, error) {
	eventSourceContainer := corev1.Container{
		Name:            "main",
		Image:           r.eventSourceImage,
		ImagePullPolicy: sharedutil.GetImagePullPolicy(),
		Args:            []string{"eventsource-service"},
		Ports: []corev1.ContainerPort{
			{Name: "metrics", ContainerPort: v1alpha1.EventSourceMetricsPort},
		},
	}
	if t := eventSource.Spec.Template; t != nil && t.Container != nil {
		if err := mergo.Merge(&eventSourceContainer, t.Container, mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	lbs := make(map[string]string)
	for k, v := range labels {
		lbs[k] = v
	}
	if t := eventSource.Spec.Template; t != nil && t.Metadata != nil &&
		len(t.Metadata.Labels) > 0 {
		for k, v := range t.Metadata.Labels {
			lbs[k] = v
		}
	}

	replicas := eventSource.Spec.GetReplicas()
	spec := &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: lbs,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: lbs,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					eventSourceContainer,
				},
			},
		},
	}
	if t := eventSource.Spec.Template; t != nil {
		if t.Metadata != nil {
			spec.Template.SetAnnotations(t.Metadata.Annotations)
		}
		spec.Template.Spec.ServiceAccountName = t.ServiceAccountName
		spec.Template.Spec.Volumes = t.Volumes
		spec.Template.Spec.SecurityContext = t.SecurityContext
		spec.Template.Spec.NodeSelector = t.NodeSelector
		spec.Template.Spec.Tolerations = t.Tolerations
		spec.Template.Spec.Affinity = t.Affinity
		spec.Template.Spec.ImagePullSecrets = t.ImagePullSecrets
		spec.Template.Spec.PriorityClassName = t.PriorityClassName
		spec.Template.Spec.Priority = t.Priority
	}
	return spec, nil
}

func (r *reconciler) findLegacyDeployment(ctx context.Context, eventSource *v1alpha1.EventSource) (*appv1.Deployment, error) {
	selector := labels.SelectorFromSet(map[string]string{
		"controller":       "eventsource-controller",
		"eventsource-name": eventSource.Name,
		"owner-name":       eventSource.Name,
	})
	dl := &appv1.DeploymentList{}
	err := r.client.List(ctx, dl, &client.ListOptions{
		Namespace:     eventSource.Namespace,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}
	for _, deploy := range dl.Items {
		if metav1.IsControlledBy(&deploy, eventSource) {
			return &deploy, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func (r *reconciler) createOrUpdateService(ctx context.Context, eventSource *v1alpha1.EventSource) error {
	log := logging.FromContext(ctx)

	// Clean up legacy services
	// TODO: remove this in a future release
	legacySvc, err := r.findLegacyService(ctx, eventSource)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Errorw("error finding legacy deployment", zap.Error(err))
			return err
		}
	} else {
		_ = r.client.Delete(ctx, legacySvc)
	}

	expectedSvc := r.buildService(eventSource)
	if expectedSvc == nil {
		return nil
	}
	svcHash := expectedSvc.GetAnnotations()[v1alpha1.KeyHash]

	existingSvc := &corev1.Service{}
	needToCreat := false
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: eventSource.Namespace, Name: expectedSvc.Name}, existingSvc); err != nil {
		if apierrors.IsNotFound(err) {
			needToCreat = true
		} else {
			return fmt.Errorf("failed to find existing service, %w", err)
		}
	} else if existingSvc.GetAnnotations()[v1alpha1.KeyHash] != svcHash {
		if err := r.client.Delete(ctx, existingSvc); err != nil && !apierrors.IsNotFound(err) {
			log.Errorw("Failed to delete existing service", zap.String("service", existingSvc.Name), zap.Error(err))
			return fmt.Errorf("failed to delete existing service, %w", err)
		}
		needToCreat = true
	}
	if needToCreat {
		if err := r.client.Create(ctx, expectedSvc); err != nil {
			log.Errorw("Failed to create daemon service", zap.String("service", expectedSvc.Name), zap.Error(err))
			return fmt.Errorf("failed to create a service, %w", err)
		}
		log.Infow("Succeeded to create/recreate a service", zap.String("service", expectedSvc.Name))
	}
	return nil
}

func (r *reconciler) buildService(eventSource *v1alpha1.EventSource) *corev1.Service {
	if eventSource.Spec.Service == nil {
		return nil
	}
	if len(eventSource.Spec.Service.Ports) == 0 {
		return nil
	}
	// Use a ports copy otherwise it will update the oririnal Ports spec in EventSource
	ports := []corev1.ServicePort{}
	ports = append(ports, eventSource.Spec.Service.Ports...)

	labels := map[string]string{
		v1alpha1.KeyManagedBy:       v1alpha1.ControllerEventSource,
		v1alpha1.KeyComponent:       v1alpha1.ComponentEventSource,
		v1alpha1.KeyPartOf:          v1alpha1.Project,
		v1alpha1.KeyAppName:         eventSource.GetDeploymentName(),
		v1alpha1.KeyEventSourceName: eventSource.Name,
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventSource.GetServiceName(),
			Namespace: eventSource.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(eventSource.GetObjectMeta(), v1alpha1.EventBusGroupVersionKind),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports:     ports,
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: eventSource.Spec.Service.ClusterIP,
			Selector:  labels,
		},
	}

	lbs := make(map[string]string)
	for k, v := range labels {
		lbs[k] = v
	}
	hash := sharedutil.MustHash(svc.Spec)
	annotations := map[string]string{
		v1alpha1.KeyHash: hash,
	}

	if s := eventSource.Spec.Service.Metadata; s != nil {
		if s.Labels != nil {
			for k, v := range s.Labels {
				lbs[k] = v
			}
		}
		if s.Annotations != nil {
			for k, v := range s.Annotations {
				annotations[k] = v
			}
		}
	}

	svc.ObjectMeta.SetLabels(lbs)
	svc.ObjectMeta.SetAnnotations(annotations)
	return svc
}

func (r *reconciler) findLegacyService(ctx context.Context, eventSource *v1alpha1.EventSource) (*corev1.Service, error) {
	selector := labels.SelectorFromSet(map[string]string{
		"controller":       "eventsource-controller",
		"eventsource-name": eventSource.Name,
		"owner-name":       eventSource.Name,
	})
	sl := &corev1.ServiceList{}
	err := r.client.List(ctx, sl, &client.ListOptions{
		Namespace:     eventSource.Namespace,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}
	for _, svc := range sl.Items {
		if metav1.IsControlledBy(&svc, eventSource) {
			return &svc, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}
