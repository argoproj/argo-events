package eventsource

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
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
	"github.com/argoproj/argo-events/eventsources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

var (
	secretKeySelectorType    = reflect.TypeOf(&corev1.SecretKeySelector{})
	configMapKeySelectorType = reflect.TypeOf(&corev1.ConfigMapKeySelector{})
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
	eventBusName := "default"
	if len(eventSource.Spec.EventBusName) > 0 {
		eventBusName = eventSource.Spec.EventBusName
	}
	err := client.Get(ctx, types.NamespacedName{Namespace: eventSource.Namespace, Name: eventBusName}, eventBus)
	if err != nil {
		if apierrors.IsNotFound(err) {
			eventSource.Status.MarkDeployFailed("EventBusNotFound", "EventBus not found.")
			logger.Errorw("EventBus not found", "eventBusName", eventBusName, "error", err)
			return errors.Errorf("eventbus %s not found", eventBusName)
		}
		eventSource.Status.MarkDeployFailed("GetEventBusFailed", "Failed to get EventBus.")
		logger.Errorw("failed to get EventBus", "eventBusName", eventBusName, "error", err)
		return err
	}
	if !eventBus.Status.IsReady() {
		eventSource.Status.MarkDeployFailed("EventBusNotReady", "EventBus not ready.")
		logger.Errorw("event bus is not in ready status", "eventBusName", eventBusName, "error", err)
		return errors.New("eventbus not ready")
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
		return nil, errors.New("failed marshal eventsource spec")
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
			Name:      "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
		},
	}

	busConfigBytes, err := json.Marshal(eventBus.Status.Config)
	if err != nil {
		return nil, errors.Errorf("failed marshal event bus config: %v", err)
	}
	encodedBusConfig := base64.StdEncoding.EncodeToString(busConfigBytes)
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
			volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "auth-volume", MountPath: common.EventBusAuthFileMountPath})
			deploymentSpec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
		}
	} else {
		return nil, errors.New("unsupported event bus")
	}

	envs := deploymentSpec.Template.Spec.Containers[0].Env
	envs = append(envs, envVars...)
	deploymentSpec.Template.Spec.Containers[0].Env = envs

	envFroms := []corev1.EnvFromSource{}
	oldEnvFroms := deploymentSpec.Template.Spec.Containers[0].EnvFrom
	if len(oldEnvFroms) > 0 {
		envFroms = append(envFroms, oldEnvFroms...)
	}
	envFromSecrets := envFromSources(args.EventSource, secretKeySelectorType)
	if len(envFromSecrets) > 0 {
		envFroms = append(envFroms, envFromSecrets...)
	}
	envFromConfigMaps := envFromSources(args.EventSource, configMapKeySelectorType)
	if len(envFromConfigMaps) > 0 {
		envFroms = append(envFroms, envFromConfigMaps...)
	}
	deploymentSpec.Template.Spec.Containers[0].EnvFrom = envFroms
	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    args.EventSource.Namespace,
			GenerateName: fmt.Sprintf("%s-eventsource-", args.EventSource.Name),
			Labels:       args.Labels,
		},
		Spec: *deploymentSpec,
	}
	if err := controllerscommon.SetObjectMeta(args.EventSource, deployment, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return deployment, nil
}

func buildDeploymentSpec(args *AdaptorArgs) (*appv1.DeploymentSpec, error) {
	singleReplica := int32(1)
	replicas := singleReplica
	if args.EventSource.Spec.Replica != nil {
		replicas = *args.EventSource.Spec.Replica
	}
	if replicas < singleReplica {
		replicas = singleReplica
	}
	eventSourceContainer := corev1.Container{
		Image:           args.Image,
		ImagePullPolicy: corev1.PullAlways,
	}
	if args.EventSource.Spec.Template.Container != nil {
		if err := mergo.Merge(&eventSourceContainer, args.EventSource.Spec.Template.Container, mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	eventSourceContainer.Name = "main"
	spec := &appv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: args.Labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: args.Labels,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: args.EventSource.Spec.Template.ServiceAccountName,
				Containers: []corev1.Container{
					eventSourceContainer,
				},
				Volumes:         args.EventSource.Spec.Template.Volumes,
				SecurityContext: args.EventSource.Spec.Template.SecurityContext,
				NodeSelector:    args.EventSource.Spec.Template.NodeSelector,
			},
		},
	}
	allEventTypes := eventsources.GetEventingServers(args.EventSource)
	recreateTypes := make(map[apicommon.EventSourceType]bool)
	for _, esType := range apicommon.RecreateStrategyEventSources {
		recreateTypes[esType] = true
	}
	recreates := 0
	for eventType := range allEventTypes {
		if _, ok := recreateTypes[eventType]; ok {
			recreates++
			break
		}
	}
	if recreates > 0 {
		spec.Replicas = &singleReplica
		spec.Strategy = appv1.DeploymentStrategy{
			Type: appv1.RecreateDeploymentStrategyType,
		}
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
			Labels:    args.Labels,
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

func labelSelector(labelMap map[string]string) labels.Selector {
	return labels.SelectorFromSet(labelMap)
}

// envFromSources returns list of EnvFromSource of an EventSource
func envFromSources(eventSource *v1alpha1.EventSource, t reflect.Type) []corev1.EnvFromSource {
	result := []corev1.EnvFromSource{}
	v := reflect.ValueOf(&eventSource.Spec).Elem()
	for j := 0; j < v.NumField(); j++ {
		f := v.Field(j)
		if f.Kind() == reflect.Map && !f.IsNil() {
			iter := f.MapRange()
			for iter.Next() {
				val := iter.Value().Interface()
				froms := envFromSecretsOrConfigMaps(val, t)
				result = append(result, froms...)
			}
		}
	}
	// Uniq
	r := []corev1.EnvFromSource{}
	keys := make(map[string]bool)
	for _, e := range result {
		var entry string
		switch t {
		case secretKeySelectorType:
			entry = e.SecretRef.Name
		case configMapKeySelectorType:
			entry = e.ConfigMapRef.Name
		default:
		}
		if _, value := keys[entry]; !value {
			keys[entry] = true
			r = append(r, e)
		}
	}
	return r
}

func envFromSecretsOrConfigMaps(source interface{}, t reflect.Type) []corev1.EnvFromSource {
	result := []corev1.EnvFromSource{}
	value := reflect.ValueOf(source)
	if value.Kind() == reflect.Ptr {
		value = reflect.Indirect(value)
	}
	if value.Kind() != reflect.Struct {
		return result
	}
	structType := value.Type()
	for i := 0; i < structType.NumField(); i++ {
		f := structType.Field(i)

		if f.Type == t {
			v := value.FieldByName(f.Name)
			if !v.IsNil() {
				switch t {
				case secretKeySelectorType:
					selector := v.Interface().(*corev1.SecretKeySelector)
					result = append(result, common.GenerateEnvFromSecretSpec(selector))
				case configMapKeySelectorType:
					selector := v.Interface().(*corev1.ConfigMapKeySelector)
					result = append(result, common.GenerateEnvFromConfigMapSpec(selector))
				default:
				}
			}
		}
	}
	return result
}
