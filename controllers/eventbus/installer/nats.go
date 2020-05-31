package installer

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj/argo-events/common"
	controllerscommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"

	"github.com/go-logr/logr"
)

const (
	clientPort  = int32(4222)
	clusterPort = int32(6222)
	monitorPort = int32(8222)

	authStrategyAnnoKey = "strategy"
	clientAuthSecretKey = "client-auth"
	serverAuthSecretKey = "auth"
	serverConfigMapKey  = "nats-server-config"
)

// natsInstaller is used create a NATS installation.
type natsInstaller struct {
	client   client.Client
	eventBus *v1alpha1.EventBus
	image    string
	labels   map[string]string
	logger   logr.Logger
}

// NewNATSInstaller returns a new NATS installer
func NewNATSInstaller(client client.Client, eventBus *v1alpha1.EventBus, image string, labels map[string]string, logger logr.Logger) Installer {
	return &natsInstaller{
		client:   client,
		eventBus: eventBus,
		image:    image,
		labels:   labels,
		logger:   logger,
	}
}

// Install creats a StatefulSet and a Service for NATS
func (i *natsInstaller) Install() (*v1alpha1.BusConfig, error) {
	natsObj := i.eventBus.Spec.NATS
	if natsObj == nil || natsObj.Native == nil {
		return nil, errors.New("invalid request")
	}
	ctx := context.Background()
	svc, err := i.createService(ctx)
	if err != nil {
		return nil, err
	}
	i.eventBus.Status.MarkServiceCreated("Succeeded", "Succeed to sync the service")
	cm, err := i.createServerAuthConfigMap(ctx)
	if err != nil {
		return nil, err
	}
	authStrategy := natsObj.Native.Auth
	if authStrategy == "" {
		// default to token auth
		authStrategy = v1alpha1.AuthStrategyToken
	}
	serverAuthSecret, clientAuthSecret, err := i.createAuthSecrets(ctx, authStrategy)
	if err != nil {
		return nil, err
	}
	if err := i.createStatefulSet(ctx, svc.Name, cm.Name, serverAuthSecret.Name); err != nil {
		return nil, err
	}
	i.eventBus.Status.MarkDeployed("Succeeded", "StatefulSet is synced")
	i.eventBus.Status.MarkConfigured()
	return &v1alpha1.BusConfig{
		NATS: &v1alpha1.NATSConfig{
			URL:  fmt.Sprintf("nats://%s:%s", generateServiceName(i.eventBus), strconv.Itoa(int(clientPort))),
			Auth: authStrategy,
			AccessSecret: corev1.SecretKeySelector{
				Key: clientAuthSecretKey,
				LocalObjectReference: corev1.LocalObjectReference{
					Name: clientAuthSecret.Name,
				},
			},
		},
	}, nil
}

// Create a service
func (i *natsInstaller) createService(ctx context.Context) (*corev1.Service, error) {
	log := i.logger
	svc, err := i.getService(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkServiceNotCreated("GetServiceFailed", "Get existing service failed")
		log.Error(err, "error getting existing service")
		return nil, err
	}
	expectedSvc, err := i.buildService()
	if err != nil {
		i.eventBus.Status.MarkServiceNotCreated("BuildServiceFailed", "Failed to build a service spec")
		log.Error(err, "error building service spec")
		return nil, err
	}
	if svc != nil {
		// TODO: potential issue here - if service spec is updated manually, reconciler will not change it back.
		// Revisit it later to see if it is needed to compare the spec.
		if svc.Annotations != nil && svc.Annotations[common.AnnotationResourceSpecHash] != expectedSvc.Annotations[common.AnnotationResourceSpecHash] {
			svc.Spec = expectedSvc.Spec
			svc.Annotations[common.AnnotationResourceSpecHash] = expectedSvc.Annotations[common.AnnotationResourceSpecHash]
			err = i.client.Update(ctx, svc)
			if err != nil {
				i.eventBus.Status.MarkServiceNotCreated("UpdateServiceFailed", "Failed to update existing service")
				log.Error(err, "error updating existing service")
				return nil, err
			}
			log.Info("service is updated", "serviceName", svc.Name)
		}
		return svc, nil
	}
	err = i.client.Create(ctx, expectedSvc)
	if err != nil {
		i.eventBus.Status.MarkServiceNotCreated("CreateFailed", "Failed to create service")
		log.Error(err, "error creating a service")
		return nil, err
	}
	log.Info("service is created", "serviceName", expectedSvc.Name)
	return expectedSvc, nil
}

//Create a Configmap for NATS config
func (i *natsInstaller) createServerAuthConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	log := i.logger
	cm, err := i.getConfigMap(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetConfigMapFailed", "Failed to get existing configmap")
		log.Error(err, "error getting existing configmap")
		return nil, err
	}
	expectedCm, err := i.buildConfigMap()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildConfigMapFailed", "Failed to build a configmap spec")
		log.Error(err, "error building configmap spec")
		return nil, err
	}
	if cm != nil {
		// TODO: Potential issue about comparing hash
		if cm.Annotations != nil && cm.Annotations[common.AnnotationResourceSpecHash] != expectedCm.Annotations[common.AnnotationResourceSpecHash] {
			cm.Data = expectedCm.Data
			cm.Annotations[common.AnnotationResourceSpecHash] = expectedCm.Annotations[common.AnnotationResourceSpecHash]
			err := i.client.Update(ctx, cm)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateConfigMapFailed", "Failed to update existing configmap")
				log.Error(err, "error updating configmap")
				return nil, err
			}
			log.Info("updated configmap", "configmapName", cm.Name)
		}
		return cm, nil
	}
	err = i.client.Create(ctx, expectedCm)
	if err != nil {
		i.eventBus.Status.MarkServiceNotCreated("CreateConfigMapFailed", "Failed to create configmap")
		log.Error(err, "error creating a configmap")
		return nil, err
	}
	log.Info("created configmap", "configmapName", expectedCm.Name)
	return expectedCm, nil
}

// create server and client auth secrets
func (i *natsInstaller) createAuthSecrets(ctx context.Context, strategy v1alpha1.AuthStrategy) (*corev1.Secret, *corev1.Secret, error) {
	log := i.logger
	sSecret, err := i.getServerAuthSecret(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetServerAuthSecretFailed", "Failed to get existing server auth secret")
		log.Error(err, "error getting existing server auth secret")
		return nil, nil, err
	}
	cSecret, err := i.getClientAuthSecret(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetClientAuthSecretFailed", "Failed to get existing client auth secret")
		log.Error(err, "error getting existing client auth secret")
		return nil, nil, err
	}
	if sSecret != nil && cSecret != nil && sSecret.Annotations != nil && cSecret.Annotations != nil {
		if sSecret.Annotations[authStrategyAnnoKey] == string(strategy) && cSecret.Annotations[authStrategyAnnoKey] == string(strategy) {
			// If the secrets are already existing, and strategy didn't change, reuse them without updating.
			return sSecret, cSecret, nil
		}
	}
	switch strategy {
	case v1alpha1.AuthStrategyToken:
		token := generateToken(64)
		serverAuthText := fmt.Sprintf(`authorization {
  token: "%s"
}`, token)
		clientAuthText := fmt.Sprintf("token=%s", token)

		expectedSSecret, err := i.buildServerAuthSecret(strategy, serverAuthText)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildServerAuthSecretFailed", "Failed to build a server auth secret spec")
			log.Error(err, "error building server auth secret spec")
			return nil, nil, err
		}
		if sSecret == nil {
			err = i.client.Create(ctx, expectedSSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("CreateServerAuthSecretFailed", "Failed to create a server auth secret")
				log.Error(err, "error creating server auth secret")
				return nil, nil, err
			}
			log.Info("created server auth secret", "serverAuthSecretName", expectedSSecret.Name)
		} else {
			sSecret.Data = expectedSSecret.Data
			err = i.client.Update(ctx, sSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateServerAuthSecretFailed", "Failed to update the server auth secret")
				log.Error(err, "error updating server auth secret")
				return nil, nil, err
			}
			log.Info("updated server auth secret", "serverAuthSecretName", sSecret.Name)
		}

		expectedCSecret, err := i.buildClientAuthSecret(strategy, clientAuthText)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildClientAuthSecretFailed", "Failed to build a client auth secret spec")
			log.Error(err, "error building client auth secret spec")
			return nil, nil, err
		}
		if cSecret == nil {
			err = i.client.Create(ctx, expectedCSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("CreateClientAuthSecretFailed", "Failed to create a client auth secret")
				log.Error(err, "error creating client auth secret")
				return nil, nil, err
			}
			log.Info("created client auth secret", "clientAuthSecretName", expectedCSecret.Name)
		} else {
			cSecret.Data = expectedCSecret.Data
			err = i.client.Update(ctx, cSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateClientAuthSecretFailed", "Failed to update the client auth secret")
				log.Error(err, "error updating client auth secret")
				return nil, nil, err
			}
			log.Info("updated client auth secret", "clientAuthSecretName", cSecret.Name)
		}
		return expectedSSecret, expectedCSecret, nil
	default:
		i.eventBus.Status.MarkDeployFailed("UnsupportedAuthStrategy", "Unsupported auth strategy")
		return nil, nil, errors.New("unsupported auth strategy")
	}
}

// Create a StatefulSet
func (i *natsInstaller) createStatefulSet(ctx context.Context, serviceName, configmapName, authSecretName string) error {
	log := i.logger
	ss, err := i.getStatefulSet(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetStatefulSetFailed", "Failed to get existing statefulset")
		log.Error(err, "error getting existing statefulset")
		return err
	}
	expectedSs, err := i.buildStatefulSet(serviceName, configmapName, authSecretName)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildStatefulSetFailed", "Failed to build a statefulset spec")
		log.Error(err, "error building statefulset spec")
		return err
	}
	if ss != nil {
		// TODO: Potential issue here - if statefulset spec is updated manually, reconciler will not change it back.
		// Revisit it later to see if it is needed to compare the spec.
		if ss.Annotations != nil && ss.Annotations[common.AnnotationResourceSpecHash] != expectedSs.Annotations[common.AnnotationResourceSpecHash] {
			ss.Spec = expectedSs.Spec
			ss.Annotations[common.AnnotationResourceSpecHash] = expectedSs.Annotations[common.AnnotationResourceSpecHash]
			err := i.client.Update(ctx, ss)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateStatefulSetFailed", "Failed to update existing statefulset")
				log.Error(err, "error updating statefulset")
				return err
			}
			log.Info("statefulset is updated", "statefulsetName", ss.Name)
		}
	} else {
		err := i.client.Create(ctx, expectedSs)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("CreateStatefulSetFailed", "Failed to create a statefulset")
			log.Error(err, "error creating a statefulset")
			return err
		}
		log.Info("statefulset is created", "statefulsetName", expectedSs.Name)
	}
	return nil
}

// buildService builds a Service for NATS
func (i *natsInstaller) buildService() (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateServiceName(i.eventBus),
			Namespace: i.eventBus.Namespace,
			Labels:    i.labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{Name: "client", Port: clientPort},
				{Name: "cluster", Port: clusterPort},
				{Name: "monitor", Port: monitorPort},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: i.labels,
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return svc, nil
}

// buildConfigMap builds a ConfigMap for NATS configuration
func (i *natsInstaller) buildConfigMap() (*corev1.ConfigMap, error) {
	routes := ""
	size := i.eventBus.Spec.NATS.Native.Size
	if size <= 0 {
		size = 1
	}
	ssName := generateStatefulSetName(i.eventBus)
	svcName := generateServiceName(i.eventBus)
	for j := 0; j < size; j++ {
		routes += fmt.Sprintf("\n    nats://%s-%s.%s.%s.svc:%s", ssName, strconv.Itoa(j), svcName, i.eventBus.Namespace, strconv.Itoa(int(clusterPort)))
	}
	conf := fmt.Sprintf(`pid_file: "/var/run/nats/nats.pid"
http: %s
cluster {
  port: %s
  include ./auth.conf
  routes: [%s
  ]
  cluster_advertise: $CLUSTER_ADVERTISE
  connect_retries: 30
}`, strconv.Itoa(int(monitorPort)), strconv.Itoa(int(clusterPort)), routes)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateConfigMapName(i.eventBus),
			Labels:    i.labels,
		},
		Data: map[string]string{
			serverConfigMapKey: conf,
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, cm, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return cm, nil
}

// buildServerAuthSecret builds a secret for NATS auth config
// Parameter - authStrategy: will be added to annoations
// Parameter - secret
// Example:
//
// authorization {
//   token: "abcd1234"
// }
func (i *natsInstaller) buildServerAuthSecret(authStrategy v1alpha1.AuthStrategy, secret string) (*corev1.Secret, error) {
	encodedAuthConf := base64.StdEncoding.EncodeToString([]byte(secret))
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   i.eventBus.Namespace,
			Name:        generateServerAuthSecretName(i.eventBus),
			Labels:      serverAuthLabels(i.labels),
			Annotations: map[string]string{authStrategyAnnoKey: string(authStrategy)},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			serverAuthSecretKey: []byte(encodedAuthConf),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, s, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return s, nil
}

// buildClientAuthSecret builds a secret for NATS client auth
func (i *natsInstaller) buildClientAuthSecret(authStrategy v1alpha1.AuthStrategy, secret string) (*corev1.Secret, error) {
	encodedAuthConf := base64.StdEncoding.EncodeToString([]byte(secret))
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   i.eventBus.Namespace,
			Name:        generateClientAuthSecretName(i.eventBus),
			Labels:      clientAuthLabels(i.labels),
			Annotations: map[string]string{authStrategyAnnoKey: string(authStrategy)},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			clientAuthSecretKey: []byte(encodedAuthConf),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, s, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return s, nil
}

// buildStatefulSet builds a StatefulSet for NATS
func (i *natsInstaller) buildStatefulSet(serviceName, configmapName, authSecretName string) (*appv1.StatefulSet, error) {
	ss := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateStatefulSetName(i.eventBus),
			Labels:    i.labels,
		},
		// Use provided serviceName, configMapName and secretName to build the spec
		// to avoid issues when naming convention changes
		Spec: i.buildStatefulSetSpec(serviceName, configmapName, authSecretName),
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, ss, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return ss, nil
}

func (i *natsInstaller) buildStatefulSetSpec(serviceName, configmapName, authSecretName string) appv1.StatefulSetSpec {
	size := int32(i.eventBus.Spec.NATS.Native.Size)
	if size == 0 {
		size = 1
	}
	terminationGracePeriodSeconds := int64(60)
	return appv1.StatefulSetSpec{
		Replicas:    &size,
		ServiceName: serviceName,
		Selector: &metav1.LabelSelector{
			MatchLabels: i.labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: i.labels,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							Projected: &corev1.ProjectedVolumeSource{
								Sources: []corev1.VolumeProjection{
									{
										ConfigMap: &corev1.ConfigMapProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configmapName,
											},
											Items: []corev1.KeyToPath{
												{
													Key:  serverConfigMapKey,
													Path: "nats.conf",
												},
											},
										},
									},
									{
										Secret: &corev1.SecretProjection{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: authSecretName,
											},
											Items: []corev1.KeyToPath{
												{
													Key:  serverAuthSecretKey,
													Path: "auth.conf",
												},
											},
										},
									},
								},
							},
						},
					},
					{
						Name: "pid",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				Containers: []corev1.Container{
					{
						Name:  "nats",
						Image: i.image,
						Ports: []corev1.ContainerPort{
							{Name: "client", ContainerPort: clientPort},
							{Name: "cluster", ContainerPort: clusterPort},
							{Name: "monitor", ContainerPort: monitorPort},
						},
						Command: []string{"/nats-server", "--config", "/etc/nats-config/nats.conf"},
						Env: []corev1.EnvVar{
							{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
							{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
							{Name: "CLUSTER_ADVERTISE", Value: "$(POD_NAME)." + generateServiceName(i.eventBus) + ".$(POD_NAMESPACE).svc"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config-volume", MountPath: "/etc/nats-config"},
							{Name: "pid", MountPath: "/var/run/nats"},
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(monitorPort)),
								},
							},
							InitialDelaySeconds: 10,
							TimeoutSeconds:      5,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/",
									Port: intstr.FromInt(int(monitorPort)),
								},
							},
							InitialDelaySeconds: 10,
							TimeoutSeconds:      5,
						},
						Lifecycle: &corev1.Lifecycle{
							PreStop: &corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{"/bin/sh", "-c", "/nats-server -sl=ldm=/var/run/nats/nats.pid && /bin/sleep 60"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (i *natsInstaller) getService(ctx context.Context) (*corev1.Service, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	sl := &corev1.ServiceList{}
	err := i.client.List(ctx, sl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(i.labels),
	})
	if err != nil {
		return nil, err
	}
	for _, svc := range sl.Items {
		if metav1.IsControlledBy(&svc, i.eventBus) {
			return &svc, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func (i *natsInstaller) getConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	cml := &corev1.ConfigMapList{}
	err := i.client.List(ctx, cml, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(i.labels),
	})
	if err != nil {
		return nil, err
	}
	for _, cm := range cml.Items {
		if metav1.IsControlledBy(&cm, i.eventBus) {
			return &cm, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

// get server auth secret
func (i *natsInstaller) getServerAuthSecret(ctx context.Context) (*corev1.Secret, error) {
	sl := &corev1.SecretList{}
	err := i.client.List(ctx, sl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(serverAuthLabels(i.labels)),
	})
	if err != nil {
		return nil, err
	}
	for _, s := range sl.Items {
		if metav1.IsControlledBy(&s, i.eventBus) {
			return &s, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

// get client auth secret
func (i *natsInstaller) getClientAuthSecret(ctx context.Context) (*corev1.Secret, error) {
	sl := &corev1.SecretList{}
	err := i.client.List(ctx, sl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(clientAuthLabels(i.labels)),
	})
	if err != nil {
		return nil, err
	}
	for _, s := range sl.Items {
		if metav1.IsControlledBy(&s, i.eventBus) {
			return &s, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
}

func (i *natsInstaller) getStatefulSet(ctx context.Context) (*appv1.StatefulSet, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	ssl := &appv1.StatefulSetList{}
	err := i.client.List(ctx, ssl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(i.labels),
	})
	if err != nil {
		return nil, err
	}
	for _, ss := range ssl.Items {
		if metav1.IsControlledBy(&ss, i.eventBus) {
			return &ss, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{}, "")
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

func serverAuthLabels(given map[string]string) map[string]string {
	result := map[string]string{"server-auth-secret": "yes"}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func clientAuthLabels(given map[string]string) map[string]string {
	result := map[string]string{"client-auth-secret": "yes"}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func labelSelector(labelMap map[string]string) labels.Selector {
	return labels.SelectorFromSet(labelMap)
}

func generateServiceName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-svc", eventBus.Name)
}

func generateConfigMapName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-nats-%s-configmap", eventBus.Name)
}

func generateServerAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-nats-%s-server", eventBus.Name)
}

func generateClientAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-nats-%s-client", eventBus.Name)
}

func generateStatefulSetName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-nats-%s", eventBus.Name)
}
