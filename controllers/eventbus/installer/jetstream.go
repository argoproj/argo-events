package installer

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"go.uber.org/zap"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	jsClientPort  = int32(4222)
	jsClusterPort = int32(6222)
	jsMonitorPort = int32(8222)
	jsMetricsPort = int32(7777)
)

var (
	//go:embed assets/jetstream/*
	jetStremAssets embed.FS
)

type jetStreamInstaller struct {
	client   client.Client
	eventBus *v1alpha1.EventBus
	config   *controllers.GlobalConfig
	labels   map[string]string
	logger   *zap.SugaredLogger
}

func NewJetStreamInstaller(client client.Client, eventBus *v1alpha1.EventBus, config *controllers.GlobalConfig, labels map[string]string, logger *zap.SugaredLogger) Installer {
	return &jetStreamInstaller{
		client:   client,
		eventBus: eventBus,
		config:   config,
		labels:   labels,
		logger:   logger.With("eventbus", eventBus.Name),
	}
}

func (r *jetStreamInstaller) Install(ctx context.Context) (*v1alpha1.BusConfig, error) {
	if js := r.eventBus.Spec.JetStream; js == nil {
		return nil, fmt.Errorf("invalid jetstream eventbus spec")
	}
	if err := r.createAuthSecrets(ctx); err != nil {
		r.logger.Errorw("failed to create jetstream auth secrets", zap.Error(err))
		r.eventBus.Status.MarkDeployFailed("JetStreamAuthSecretsFailed", err.Error())
		return nil, err
	}
	if err := r.createConfigMap(ctx); err != nil {
		r.logger.Errorw("failed to create jetstream ConfigMap", zap.Error(err))
		r.eventBus.Status.MarkDeployFailed("JetStreamConfigMapFailed", err.Error())
		return nil, err
	}
	if err := r.createService(ctx); err != nil {
		r.logger.Errorw("failed to create jetstream Service", zap.Error(err))
		r.eventBus.Status.MarkDeployFailed("JetStreamServiceFailed", err.Error())
		return nil, err
	}
	if err := r.createStatefulSet(ctx); err != nil {
		r.logger.Errorw("failed to create jetstream StatefulSet", zap.Error(err))
		r.eventBus.Status.MarkDeployFailed("JetStreamStatefulSetFailed", err.Error())
		return nil, err
	}
	r.eventBus.Status.MarkDeployed("Succeeded", "JetStream is deployed")
	return &v1alpha1.BusConfig{
		JetStream: &v1alpha1.JetStreamConfig{
			URL: fmt.Sprintf("nats://%s.%s.svc.cluster.local:%s", generateJetStreamServiceName(r.eventBus), r.eventBus.Namespace, strconv.Itoa(int(jsClientPort))),
			Auth: &v1alpha1.JetStreamAuth{
				Token: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: generateJetStreamClientAuthSecretName(r.eventBus),
					},
					Key: common.JetStreamClientAuthSecretKey,
				},
			},
		},
	}, nil
}

// buildJetStreamService builds a Service for Jet Stream
func (r *jetStreamInstaller) buildJetStreamServiceSpec() corev1.ServiceSpec {
	return corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{Name: "tcp-client", Port: jsClientPort},
			{Name: "cluster", Port: jsClusterPort},
			{Name: "metrics", Port: jsMetricsPort},
			{Name: "monitor", Port: jsMonitorPort},
		},
		Type:                     corev1.ServiceTypeClusterIP,
		ClusterIP:                corev1.ClusterIPNone,
		PublishNotReadyAddresses: true,
		Selector:                 r.labels,
	}
}

func (r *jetStreamInstaller) createService(ctx context.Context) error {
	spec := r.buildJetStreamServiceSpec()
	hash := common.MustHash(spec)
	obj := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.eventBus.Namespace,
			Name:      generateJetStreamServiceName(r.eventBus),
			Labels:    r.labels,
			Annotations: map[string]string{
				common.AnnotationResourceSpecHash: hash,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.eventBus.GetObjectMeta(), v1alpha1.SchemaGroupVersionKind),
			},
		},
		Spec: spec,
	}
	old := &corev1.Service{}
	if err := r.client.Get(ctx, client.ObjectKeyFromObject(obj), old); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.client.Create(ctx, obj); err != nil {
				return fmt.Errorf("failed to create jetstream service, err: %w", err)
			}
			r.logger.Info("created jetstream service successfully")
			return nil
		} else {
			return fmt.Errorf("failed to check if jetstream service is existing, err: %w", err)
		}
	}
	if old.GetAnnotations()[common.AnnotationResourceSpecHash] != hash {
		old.Annotations[common.AnnotationResourceSpecHash] = hash
		old.Spec = spec
		if err := r.client.Update(ctx, old); err != nil {
			return fmt.Errorf("failed to update jetstream service, err: %w", err)
		}
		r.logger.Info("updated jetstream service successfully")
	}
	return nil
}

func (r *jetStreamInstaller) createStatefulSet(ctx context.Context) error {
	jsVersion, err := r.config.GetJetStreamVersion(r.eventBus.Spec.JetStream.Version)
	if err != nil {
		return fmt.Errorf("failed to get jetstream version, err: %w", err)
	}
	spec := r.buildStatefulSetSpec(jsVersion)
	hash := common.MustHash(spec)
	obj := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.eventBus.Namespace,
			Name:      generateJetStreamStatefulSetName(r.eventBus),
			Labels:    r.labels,
			Annotations: map[string]string{
				common.AnnotationResourceSpecHash: hash,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.eventBus.GetObjectMeta(), v1alpha1.SchemaGroupVersionKind),
			},
		},
		Spec: spec,
	}
	old := &appv1.StatefulSet{}
	if err := r.client.Get(ctx, client.ObjectKeyFromObject(obj), old); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.client.Create(ctx, obj); err != nil {
				return fmt.Errorf("failed to create jetstream statefulset, err: %w", err)
			}
			r.logger.Info("created jetstream statefulset successfully")
			return nil
		} else {
			return fmt.Errorf("failed to check if jetstream statefulset is existing, err: %w", err)
		}
	}
	if old.GetAnnotations()[common.AnnotationResourceSpecHash] != hash {
		old.Annotations[common.AnnotationResourceSpecHash] = hash
		old.Spec = spec
		if err := r.client.Update(ctx, old); err != nil {
			return fmt.Errorf("failed to update jetstream statefulset, err: %w", err)
		}
		r.logger.Info("updated jetstream statefulset successfully")
	}
	return nil
}

func (r *jetStreamInstaller) buildStatefulSetSpec(jsVersion *controllers.JetStreamVersion) appv1.StatefulSetSpec {
	js := r.eventBus.Spec.JetStream
	replicas := int32(js.GetReplicas())
	podTemplateLabels := make(map[string]string)
	if js.Metadata != nil &&
		len(js.Metadata.Labels) > 0 {
		for k, v := range js.Metadata.Labels {
			podTemplateLabels[k] = v
		}
	}
	for k, v := range r.labels {
		podTemplateLabels[k] = v
	}
	var jsContainerPullPolicy, reloaderContainerPullPolicy, metricsContainerPullPolicy corev1.PullPolicy
	var jsContainerSecurityContext, reloaderContainerSecurityContext, metricsContainerSecurityContext *corev1.SecurityContext
	if js.ContainerTemplate != nil {
		jsContainerPullPolicy = js.ContainerTemplate.ImagePullPolicy
		jsContainerSecurityContext = js.ContainerTemplate.SecurityContext
	}
	if js.ReloaderContainerTemplate != nil {
		reloaderContainerPullPolicy = js.ReloaderContainerTemplate.ImagePullPolicy
		reloaderContainerSecurityContext = js.ReloaderContainerTemplate.SecurityContext
	}
	if js.MetricsContainerTemplate != nil {
		metricsContainerPullPolicy = js.MetricsContainerTemplate.ImagePullPolicy
		metricsContainerSecurityContext = js.MetricsContainerTemplate.SecurityContext
	}
	shareProcessNamespace := true
	terminationGracePeriodSeconds := int64(60)
	spec := appv1.StatefulSetSpec{
		PodManagementPolicy: appv1.ParallelPodManagement,
		Replicas:            &replicas,
		ServiceName:         generateJetStreamServiceName(r.eventBus),
		Selector: &metav1.LabelSelector{
			MatchLabels: r.labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podTemplateLabels,
			},
			Spec: corev1.PodSpec{
				NodeSelector:                  js.NodeSelector,
				Tolerations:                   js.Tolerations,
				SecurityContext:               js.SecurityContext,
				ImagePullSecrets:              js.ImagePullSecrets,
				PriorityClassName:             js.PriorityClassName,
				Priority:                      js.Priority,
				ServiceAccountName:            js.ServiceAccountName,
				Affinity:                      js.Affinity,
				ShareProcessNamespace:         &shareProcessNamespace,
				TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				Volumes: []corev1.Volume{
					{Name: "pid", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
					{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{
										ConfigMap: &corev1.ConfigMapProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: generateJetStreamConfigMapName(r.eventBus),
											},
											Items: []corev1.KeyToPath{
												{
													Key:  common.JetStreamConfigMapKey,
													Path: "nats-js.conf",
												},
											},
										},
									},
									{
										Secret: &corev1.SecretProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: generateJetStreamServerAuthSecretName(r.eventBus),
											},
											Items: []corev1.KeyToPath{
												{
													Key:  common.JetStreamServerAuthSecretKey,
													Path: "auth.conf",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:            "main",
						Image:           jsVersion.NatsImage,
						ImagePullPolicy: jsContainerPullPolicy,
						Ports: []corev1.ContainerPort{
							{Name: "client", ContainerPort: jsClientPort},
							{Name: "cluster", ContainerPort: jsClusterPort},
							{Name: "monitor", ContainerPort: jsMonitorPort},
						},
						Command: []string{jsVersion.StartCommand, "--config", "/etc/nats-config/nats-js.conf"},
						Env: []corev1.EnvVar{
							{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
							{Name: "SERVER_NAME", Value: "$(POD_NAME)"},
							{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
							{Name: "CLUSTER_ADVERTISE", Value: "$(POD_NAME)." + generateJetStreamServiceName(r.eventBus) + ".$(POD_NAMESPACE).svc.cluster.local"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config-volume", MountPath: "/etc/nats-config"},
							{Name: "pid", MountPath: "/var/run/nats"},
						},
						SecurityContext: jsContainerSecurityContext,
						StartupProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.FromInt(int(jsMonitorPort)),
								},
							},
							FailureThreshold:    30,
							InitialDelaySeconds: 10,
							TimeoutSeconds:      5,
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(jsMonitorPort)),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       30,
							TimeoutSeconds:      5,
						},
						Lifecycle: &corev1.Lifecycle{
							PreStop: &corev1.LifecycleHandler{
								Exec: &corev1.ExecAction{
									Command: []string{jsVersion.StartCommand, "-sl=ldm=/var/run/nats/nats.pid"},
								},
							},
						},
					},
					{
						Name:            "reloader",
						Image:           jsVersion.ConfigReloaderImage,
						ImagePullPolicy: reloaderContainerPullPolicy,
						SecurityContext: reloaderContainerSecurityContext,
						Command:         []string{"nats-server-config-reloader", "-pid", "/var/run/nats/nats.pid", "-config", "/etc/nats-config/nats-js.conf"},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config-volume", MountPath: "/etc/nats-config"},
							{Name: "pid", MountPath: "/var/run/nats"},
						},
					},
					{
						Name:            "metrics",
						Image:           jsVersion.MetricsExporterImage,
						ImagePullPolicy: metricsContainerPullPolicy,
						Ports: []corev1.ContainerPort{
							{Name: "metrics", ContainerPort: jsMetricsPort},
						},
						Args:            []string{"-connz", "-routez", "-subz", "-varz", "-prefix=nats", "-use_internal_server_id", "-jsz=all", fmt.Sprintf("http://localhost:%s", strconv.Itoa(int(jsMonitorPort)))},
						SecurityContext: metricsContainerSecurityContext,
					},
				},
			},
		},
	}
	if js.Metadata != nil {
		spec.Template.SetAnnotations(js.Metadata.Annotations)
	}
	if js.ContainerTemplate != nil {
		spec.Template.Spec.Containers[0].Resources = js.ContainerTemplate.Resources
	}
	if js.MetricsContainerTemplate != nil {
		spec.Template.Spec.Containers[1].Resources = js.MetricsContainerTemplate.Resources
	}
	if js.Persistence != nil {
		volMode := corev1.PersistentVolumeFilesystem
		// Default volume size
		volSize := apiresource.MustParse("20Gi")
		if js.Persistence.VolumeSize != nil {
			volSize = *js.Persistence.VolumeSize
		}
		// Default to ReadWriteOnce
		accessMode := corev1.ReadWriteOnce
		if js.Persistence.AccessMode != nil {
			accessMode = *js.Persistence.AccessMode
		}
		spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: generateJetStreamPVCName(r.eventBus),
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						accessMode,
					},
					VolumeMode:       &volMode,
					StorageClassName: js.Persistence.StorageClassName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: volSize,
						},
					},
				},
			},
		}
		volumeMounts := spec.Template.Spec.Containers[0].VolumeMounts
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: generateJetStreamPVCName(r.eventBus), MountPath: "/data/jetstream"})
		spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	} else {
		// When the POD is runasnonroot, it can not create the dir /data/jetstream
		// Use an emptyDirVolume
		emptyDirVolName := "js-data"
		volumes := spec.Template.Spec.Volumes
		volumes = append(volumes, corev1.Volume{Name: emptyDirVolName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
		spec.Template.Spec.Volumes = volumes
		volumeMounts := spec.Template.Spec.Containers[0].VolumeMounts
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: emptyDirVolName, MountPath: "/data/jetstream"})
		spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	}
	return spec
}

func (r *jetStreamInstaller) createAuthSecrets(ctx context.Context) error {
	token := common.RandomString(24)
	sysPassword := common.RandomString(24)
	authTpl := template.Must(template.ParseFS(jetStremAssets, "assets/jetstream/server-auth.conf"))
	var authTplOutput bytes.Buffer
	if err := authTpl.Execute(&authTplOutput, struct {
		Token       string
		SysPassword string
	}{
		Token:       token,
		SysPassword: sysPassword,
	}); err != nil {
		return fmt.Errorf("failed to parse nats auth template, error: %w", err)
	}

	serverAuthObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.eventBus.Namespace,
			Name:      generateJetStreamServerAuthSecretName(r.eventBus),
			Labels:    r.labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.eventBus.GetObjectMeta(), v1alpha1.SchemaGroupVersionKind),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			common.JetStreamServerAuthSecretKey: authTplOutput.Bytes(),
		},
	}

	clientAuthObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.eventBus.Namespace,
			Name:      generateJetStreamClientAuthSecretName(r.eventBus),
			Labels:    r.labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.eventBus.GetObjectMeta(), v1alpha1.SchemaGroupVersionKind),
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			common.JetStreamClientAuthSecretKey: []byte(token),
		},
	}

	oldServerObjExisting, oldClientObjExisting := true, true

	oldSObj := &corev1.Secret{}
	if err := r.client.Get(ctx, client.ObjectKeyFromObject(serverAuthObj), oldSObj); err != nil {
		if apierrors.IsNotFound(err) {
			oldServerObjExisting = false
		} else {
			return fmt.Errorf("failed to check if nats server auth secret is existing, err: %w", err)
		}
	}

	oldCObj := &corev1.Secret{}
	if err := r.client.Get(ctx, client.ObjectKeyFromObject(clientAuthObj), oldCObj); err != nil {
		if apierrors.IsNotFound(err) {
			oldClientObjExisting = false
		} else {
			return fmt.Errorf("failed to check if nats client auth secret is existing, err: %w", err)
		}
	}

	if oldClientObjExisting && oldServerObjExisting { // Both existing, do nothing
		return nil
	}

	if oldClientObjExisting {
		if err := r.client.Delete(ctx, oldSObj); err != nil {
			return fmt.Errorf("failed to delete malformed nats server auth secret, err: %w", err)
		}
		r.logger.Infow("deleted malformed nats server auth secret successfully")
	}

	if oldServerObjExisting {
		if err := r.client.Delete(ctx, oldCObj); err != nil {
			return fmt.Errorf("failed to delete malformed nats client auth secret, err: %w", err)
		}
		r.logger.Infow("deleted malformed nats client auth secret successfully")
	}

	if err := r.client.Create(ctx, serverAuthObj); err != nil {
		return fmt.Errorf("failed to create nats server auth secret, err: %w", err)
	}
	r.logger.Infow("created nats server auth secret successfully")

	if err := r.client.Create(ctx, clientAuthObj); err != nil {
		return fmt.Errorf("failed to create nats client auth secret, err: %w", err)
	}
	r.logger.Infow("created nats client auth secret successfully")
	return nil
}

func (r *jetStreamInstaller) createConfigMap(ctx context.Context) error {
	data := make(map[string]string)
	svcName := generateJetStreamServiceName(r.eventBus)
	ssName := generateJetStreamStatefulSetName(r.eventBus)
	replicas := r.eventBus.Spec.JetStream.GetReplicas()
	if replicas < 3 {
		replicas = 3
	}
	routes := []string{}
	for j := 0; j < replicas; j++ {
		routes = append(routes, fmt.Sprintf("nats://%s-%s.%s.%s.svc.cluster.local:%s", ssName, strconv.Itoa(j), svcName, r.eventBus.Namespace, strconv.Itoa(int(jsClusterPort))))
	}
	settings := r.config.EventBus.JetStream.Settings
	if x := r.eventBus.Spec.JetStream.Settings; x != nil {
		settings = *x
	}

	confTpl := template.Must(template.ParseFS(jetStremAssets, "assets/jetstream/nats.conf"))
	var confTplOutput bytes.Buffer
	if err := confTpl.Execute(&confTplOutput, struct {
		ClusterName string
		MonitorPort string
		ClusterPort string
		ClientPort  string
		Routes      string
		Settings    string
	}{
		ClusterName: r.eventBus.Name,
		MonitorPort: strconv.Itoa(int(jsMonitorPort)),
		ClusterPort: strconv.Itoa(int(jsClusterPort)),
		ClientPort:  strconv.Itoa(int(jsClientPort)),
		Routes:      strings.Join(routes, ","),
		Settings:    settings,
	}); err != nil {
		return fmt.Errorf("failed to parse nats config template, error: %w", err)
	}
	data[common.JetStreamConfigMapKey] = confTplOutput.String()

	hash := common.MustHash(data)
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.eventBus.Namespace,
			Name:      generateJetStreamConfigMapName(r.eventBus),
			Labels:    r.labels,
			Annotations: map[string]string{
				common.AnnotationResourceSpecHash: hash,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.eventBus.GetObjectMeta(), v1alpha1.SchemaGroupVersionKind),
			},
		},
		Data: data,
	}
	old := &corev1.ConfigMap{}
	if err := r.client.Get(ctx, client.ObjectKeyFromObject(obj), old); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.client.Create(ctx, obj); err != nil {
				return fmt.Errorf("failed to create jetstream configmap, err: %w", err)
			}
			r.logger.Info("created jetstream configmap successfully")
			return nil
		} else {
			return fmt.Errorf("failed to check if jetstream configmap is existing, err: %w", err)
		}
	}
	if old.GetAnnotations()[common.AnnotationResourceSpecHash] != hash {
		old.Annotations[common.AnnotationResourceSpecHash] = hash
		old.Data = data
		if err := r.client.Update(ctx, old); err != nil {
			return fmt.Errorf("failed to update jetstream configmap, err: %w", err)
		}
		r.logger.Info("updated jetstream configmap successfully")
	}
	return nil
}

func (r *jetStreamInstaller) Uninstall(ctx context.Context) error {
	return r.uninstallPVCs(ctx)
}

func (r *jetStreamInstaller) uninstallPVCs(ctx context.Context) error {
	// StatefulSet doesn't clean up PVC, needs to do it separately
	// https://github.com/kubernetes/kubernetes/issues/55045
	pvcs, err := r.getPVCs(ctx)
	if err != nil {
		r.logger.Errorw("failed to get PVCs created by Nats statefulset when uninstalling", zap.Error(err))
		return err
	}
	for _, pvc := range pvcs {
		err = r.client.Delete(ctx, &pvc)
		if err != nil {
			r.logger.Errorw("failed to delete pvc when uninstalling", zap.Any("pvcName", pvc.Name), zap.Error(err))
			return err
		}
		r.logger.Infow("pvc deleted", "pvcName", pvc.Name)
	}
	return nil
}

// get PVCs created by streaming statefulset
// they have same labels as the statefulset
func (r *jetStreamInstaller) getPVCs(ctx context.Context) ([]corev1.PersistentVolumeClaim, error) {
	pvcl := &corev1.PersistentVolumeClaimList{}
	err := r.client.List(ctx, pvcl, &client.ListOptions{
		Namespace:     r.eventBus.Namespace,
		LabelSelector: labels.SelectorFromSet(r.labels),
	})
	if err != nil {
		return nil, err
	}
	return pvcl.Items, nil
}

func generateJetStreamServerAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-js-server-auth", eventBus.Name)
}

func generateJetStreamClientAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-js-client-auth", eventBus.Name)
}

func generateJetStreamServiceName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-js-svc", eventBus.Name)
}

func generateJetStreamStatefulSetName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-js", eventBus.Name)
}

func generateJetStreamConfigMapName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-js-config", eventBus.Name)
}

func generateJetStreamPVCName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-js-vol", eventBus.Name)
}
