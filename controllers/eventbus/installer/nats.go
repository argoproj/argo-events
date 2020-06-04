package installer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
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
	metricsPort = int32(7777)

	// annotation key on serverAuthSecret and clientAuthsecret
	authStrategyAnnoKey = "strategy"

	// label key on each component object
	componentLabelKey = "component"

	clientAuthSecretKey   = "client-auth"
	serverAuthSecretKey   = "auth"
	serverConfigMapKey    = "nats-server-config"
	streamingConfigMapKey = "stan-config"
)

type component string

var (
	componentServer    component = "nats-server"
	componentStreaming component = "nats-streaming"
)

// natsInstaller is used create a NATS installation.
type natsInstaller struct {
	client         client.Client
	eventBus       *v1alpha1.EventBus
	natsImage      string
	streamingImage string
	labels         map[string]string
	logger         logr.Logger
}

// NewNATSInstaller returns a new NATS installer
func NewNATSInstaller(client client.Client, eventBus *v1alpha1.EventBus, natsImage, streamingImage string, labels map[string]string, logger logr.Logger) Installer {
	return &natsInstaller{
		client:         client,
		eventBus:       eventBus,
		natsImage:      natsImage,
		streamingImage: streamingImage,
		labels:         labels,
		logger:         logger.WithName("nats"),
	}
}

// Install creats a StatefulSet and a Service for NATS
func (i *natsInstaller) Install() (*v1alpha1.BusConfig, error) {
	natsObj := i.eventBus.Spec.NATS
	if natsObj == nil || natsObj.Native == nil {
		return nil, errors.New("invalid request")
	}
	ctx := context.Background()
	svc, err := i.createServerService(ctx)
	if err != nil {
		return nil, err
	}
	cm, err := i.createServerConfigMap(ctx)
	if err != nil {
		return nil, err
	}
	// default to token auth
	defaultAuthStrategy := v1alpha1.AuthStrategyToken
	authStrategy := natsObj.Native.Auth
	if authStrategy == nil {
		authStrategy = &defaultAuthStrategy
	}
	serverAuthSecret, clientAuthSecret, err := i.createAuthSecrets(ctx, *authStrategy)
	if err != nil {
		return nil, err
	}
	if err := i.createServerStatefulSet(ctx, svc.Name, cm.Name, serverAuthSecret.Name); err != nil {
		return nil, err
	}
	// Install nats steaming
	streamingSvc, err := i.createStreamingService(ctx)
	if err != nil {
		return nil, err
	}
	streamingCm, err := i.createStreamingConfigMap(ctx)
	if err != nil {
		return nil, err
	}
	if err := i.createStreamingStatefulSet(ctx, streamingSvc.Name, streamingCm.Name); err != nil {
		return nil, err
	}
	i.eventBus.Status.MarkDeployed("Succeeded", "NATS is deployed")
	i.eventBus.Status.MarkConfigured()
	busConfig := &v1alpha1.BusConfig{
		NATS: &v1alpha1.NATSConfig{
			URL:       fmt.Sprintf("nats://%s:%s", generateServerServiceName(i.eventBus), strconv.Itoa(int(clientPort))),
			ClusterID: generateStreamingClusterID(i.eventBus),
			Auth:      *authStrategy,
		},
	}
	if *authStrategy != v1alpha1.AuthStrategyNone {
		busConfig.NATS.AccessSecret = &corev1.SecretKeySelector{
			Key: clientAuthSecretKey,
			LocalObjectReference: corev1.LocalObjectReference{
				Name: clientAuthSecret.Name,
			},
		}
	}
	return busConfig, nil
}

// Uninstall deletes those objects not handeled by cascade deletion.
func (i *natsInstaller) Uninstall() error {
	ctx := context.Background()
	return i.uninstallPVCs(ctx)
}

func (i *natsInstaller) uninstallPVCs(ctx context.Context) error {
	// StatefulSet doens't clean up PVC, needs to do it separately
	// https://github.com/kubernetes/kubernetes/issues/55045
	log := i.logger
	pvcs, err := i.getPVCs(ctx, i.componentLabels(componentStreaming))
	if err != nil {
		log.Error(err, "failed to get PVCs created by nats streaming statefulset when uninstalling")
		return err
	}
	for _, pvc := range pvcs {
		err = i.client.Delete(ctx, &pvc)
		if err != nil {
			log.Error(err, "failed to delete pvc when uninstalling", "pvcName", pvc.Name)
			return err
		}
		log.Info("pvc deleted", "pvcName", pvc.Name)
	}
	return nil
}

// Create a service for nats server
func (i *natsInstaller) createServerService(ctx context.Context) (*corev1.Service, error) {
	log := i.logger
	svc, err := i.getServerService(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetServerServiceFailed", "Get existing server service failed")
		log.Error(err, "error getting existing server service")
		return nil, err
	}
	expectedSvc, err := i.buildServerService()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildServerServiceFailed", "Failed to build a server service spec")
		log.Error(err, "error building server service spec")
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
				i.eventBus.Status.MarkDeployFailed("UpdateServerServiceFailed", "Failed to update existing server service")
				log.Error(err, "error updating existing server service")
				return nil, err
			}
			log.Info("server service is updated", "serviceName", svc.Name)
		}
		return svc, nil
	}
	err = i.client.Create(ctx, expectedSvc)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("CreateServerServiceFailed", "Failed to create server service")
		log.Error(err, "error creating a server service")
		return nil, err
	}
	log.Info("server service is created", "serviceName", expectedSvc.Name)
	return expectedSvc, nil
}

// Create a service for nats streaming
func (i *natsInstaller) createStreamingService(ctx context.Context) (*corev1.Service, error) {
	log := i.logger
	svc, err := i.getStreamingService(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetStreamingServiceFailed", "Get existing streaming service failed")
		log.Error(err, "error getting existing streaming service")
		return nil, err
	}
	expectedSvc, err := i.buildStreamingService()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildStreamingServiceFailed", "Failed to build a streaming service spec")
		log.Error(err, "error building streaming service spec")
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
				i.eventBus.Status.MarkDeployFailed("UpdateStreamingServiceFailed", "Failed to update existing streaming service")
				log.Error(err, "error updating existing streaming service")
				return nil, err
			}
			log.Info("streaming service is updated", "serviceName", svc.Name)
		}
		return svc, nil
	}
	err = i.client.Create(ctx, expectedSvc)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("CreateStreamingServiceFailed", "Failed to create a streaming service")
		log.Error(err, "error creating a streaming service")
		return nil, err
	}
	log.Info("streaming service is created", "serviceName", expectedSvc.Name)
	return expectedSvc, nil
}

//Create a Configmap for NATS config
func (i *natsInstaller) createServerConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	log := i.logger
	cm, err := i.getServerConfigMap(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetServerConfigMapFailed", "Failed to get existing server configmap")
		log.Error(err, "error getting existing server configmap")
		return nil, err
	}
	expectedCm, err := i.buildServerConfigMap()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildServerConfigMapFailed", "Failed to build a server configmap spec")
		log.Error(err, "error building server configmap spec")
		return nil, err
	}
	if cm != nil {
		// TODO: Potential issue about comparing hash
		if cm.Annotations != nil && cm.Annotations[common.AnnotationResourceSpecHash] != expectedCm.Annotations[common.AnnotationResourceSpecHash] {
			cm.Data = expectedCm.Data
			cm.Annotations[common.AnnotationResourceSpecHash] = expectedCm.Annotations[common.AnnotationResourceSpecHash]
			err := i.client.Update(ctx, cm)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateServerConfigMapFailed", "Failed to update existing server configmap")
				log.Error(err, "error updating  server configmap")
				return nil, err
			}
			log.Info("updated server configmap", "configmapName", cm.Name)
		}
		return cm, nil
	}
	err = i.client.Create(ctx, expectedCm)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("CreateServerConfigMapFailed", "Failed to create server configmap")
		log.Error(err, "error creating a server configmap")
		return nil, err
	}
	log.Info("created server configmap", "configmapName", expectedCm.Name)
	return expectedCm, nil
}

//Create a Configmap for NATS streaming config
func (i *natsInstaller) createStreamingConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	log := i.logger
	cm, err := i.getStreamingConfigMap(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetStreamingConfigMapFailed", "Failed to get existing streaming configmap")
		log.Error(err, "error getting existing streaming configmap")
		return nil, err
	}
	expectedCm, err := i.buildStreamingConfigMap()
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildStreamingConfigMapFailed", "Failed to build a streaming configmap spec")
		log.Error(err, "error building streaming configmap spec")
		return nil, err
	}
	if cm != nil {
		// TODO: Potential issue about comparing hash
		if cm.Annotations != nil && cm.Annotations[common.AnnotationResourceSpecHash] != expectedCm.Annotations[common.AnnotationResourceSpecHash] {
			cm.Data = expectedCm.Data
			cm.Annotations[common.AnnotationResourceSpecHash] = expectedCm.Annotations[common.AnnotationResourceSpecHash]
			err := i.client.Update(ctx, cm)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateStreamingConfigMapFailed", "Failed to update existing streaming configmap")
				log.Error(err, "error updating streaming configmap")
				return nil, err
			}
			log.Info("updated streaming configmap", "configmapName", cm.Name)
		}
		return cm, nil
	}
	err = i.client.Create(ctx, expectedCm)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("CreateStreamingConfigMapFailed", "Failed to create streaming configmap")
		log.Error(err, "error creating a streaming configmap")
		return nil, err
	}
	log.Info("created streaming configmap", "configmapName", expectedCm.Name)
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
	if strategy != v1alpha1.AuthStrategyNone { // Do not checkout AuthStrategyNone because it only has server auth secret
		if sSecret != nil && cSecret != nil && sSecret.Annotations != nil && cSecret.Annotations != nil {
			if sSecret.Annotations[authStrategyAnnoKey] == string(strategy) && cSecret.Annotations[authStrategyAnnoKey] == string(strategy) {
				// If the secrets are already existing, and strategy didn't change, reuse them without updating.
				return sSecret, cSecret, nil
			}
		}
	}

	switch strategy {
	case v1alpha1.AuthStrategyNone:
		// Clean up client auth secret if existing
		if cSecret != nil {
			err = i.client.Delete(ctx, cSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("DeleteClientAuthSecretFailed", "Failed to delete the client auth secret")
				log.Error(err, "error deleting client auth secret")
				return nil, nil, err
			}
			log.Info("deleted server auth secret")
		}
		if sSecret != nil && sSecret.Annotations != nil && sSecret.Annotations[authStrategyAnnoKey] == string(strategy) && len(sSecret.Data[serverAuthSecretKey]) == 0 {
			// If the server auth secret is already existing, strategy didn't change, and the secret is empty string, reuse it without updating.
			return sSecret, nil, nil
		}
		// Only create an empty server auth secret
		expectedSSecret, err := i.buildServerAuthSecret(strategy, "")
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildServerAuthSecretFailed", "Failed to build a server auth secret spec")
			log.Error(err, "error building server auth secret spec")
			return nil, nil, err
		}
		if sSecret != nil {
			sSecret.ObjectMeta.Labels = expectedSSecret.Labels
			sSecret.ObjectMeta.Annotations = expectedSSecret.Annotations
			sSecret.Data = expectedSSecret.Data
			err = i.client.Update(ctx, sSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateServerAuthSecretFailed", "Failed to update the server auth secret")
				log.Error(err, "error updating server auth secret")
				return nil, nil, err
			}
			log.Info("updated server auth secret", "serverAuthSecretName", sSecret.Name)
			return sSecret, nil, nil
		}
		err = i.client.Create(ctx, expectedSSecret)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("CreateServerAuthSecretFailed", "Failed to create a server auth secret")
			log.Error(err, "error creating server auth secret")
			return nil, nil, err
		}
		log.Info("created server auth secret", "serverAuthSecretName", expectedSSecret.Name)
		return expectedSSecret, nil, nil
	case v1alpha1.AuthStrategyToken:
		token := generateToken(64)
		serverAuthText := fmt.Sprintf(`authorization {
  token: "%s"
}`, token)
		clientAuthText := fmt.Sprintf("token=%s", token)
		// Create server auth secret
		expectedSSecret, err := i.buildServerAuthSecret(strategy, serverAuthText)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildServerAuthSecretFailed", "Failed to build a server auth secret spec")
			log.Error(err, "error building server auth secret spec")
			return nil, nil, err
		}
		returnedSSecret := expectedSSecret
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
			sSecret.ObjectMeta.Labels = expectedSSecret.Labels
			sSecret.ObjectMeta.Annotations = expectedSSecret.Annotations
			err = i.client.Update(ctx, sSecret)
			if err != nil {
				i.eventBus.Status.MarkDeployFailed("UpdateServerAuthSecretFailed", "Failed to update the server auth secret")
				log.Error(err, "error updating server auth secret")
				return nil, nil, err
			}
			log.Info("updated server auth secret", "serverAuthSecretName", sSecret.Name)
			returnedSSecret = sSecret
		}
		// create client auth secret
		expectedCSecret, err := i.buildClientAuthSecret(strategy, clientAuthText)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("BuildClientAuthSecretFailed", "Failed to build a client auth secret spec")
			log.Error(err, "error building client auth secret spec")
			return nil, nil, err
		}
		returnedCSecret := expectedCSecret
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
			returnedCSecret = cSecret
		}
		return returnedSSecret, returnedCSecret, nil
	default:
		i.eventBus.Status.MarkDeployFailed("UnsupportedAuthStrategy", "Unsupported auth strategy")
		return nil, nil, errors.New("unsupported auth strategy")
	}
}

// Create a server StatefulSet
func (i *natsInstaller) createServerStatefulSet(ctx context.Context, serviceName, configmapName, authSecretName string) error {
	log := i.logger
	ss, err := i.getServerStatefulSet(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetServerStatefulSetFailed", "Failed to get existing server statefulset")
		log.Error(err, "error getting existing server statefulset")
		return err
	}
	expectedSs, err := i.buildServerStatefulSet(serviceName, configmapName, authSecretName)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildServerStatefulSetFailed", "Failed to build a server statefulset spec")
		log.Error(err, "error building server statefulset spec")
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
				i.eventBus.Status.MarkDeployFailed("UpdateServerStatefulSetFailed", "Failed to update existing server statefulset")
				log.Error(err, "error updating statefulset")
				return err
			}
			log.Info("server statefulset is updated", "statefulsetName", ss.Name)
		}
	} else {
		err := i.client.Create(ctx, expectedSs)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("CreateServerStatefulSetFailed", "Failed to create a server statefulset")
			log.Error(err, "error creating a server statefulset")
			return err
		}
		log.Info("server statefulset is created", "statefulsetName", expectedSs.Name)
	}
	return nil
}

// Create a streaming StatefulSet
func (i *natsInstaller) createStreamingStatefulSet(ctx context.Context, serviceName, configmapName string) error {
	log := i.logger
	ss, err := i.getStreamingStatefulSet(ctx)
	if err != nil && !apierrors.IsNotFound(err) {
		i.eventBus.Status.MarkDeployFailed("GetStreamingStatefulSetFailed", "Failed to get existing streaming statefulset")
		log.Error(err, "error getting existing streaming statefulset")
		return err
	}
	expectedSs, err := i.buildStreamingStatefulSet(serviceName, configmapName)
	if err != nil {
		i.eventBus.Status.MarkDeployFailed("BuildStreamingStatefulSetFailed", "Failed to build a streaming statefulset spec")
		log.Error(err, "error building streaming statefulset spec")
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
				i.eventBus.Status.MarkDeployFailed("UpdateStreamingStatefulSetFailed", "Failed to update existing streaming statefulset")
				log.Error(err, "error updating streaming statefulset")
				return err
			}
			log.Info("streaming statefulset is updated", "statefulsetName", ss.Name)
		}
	} else {
		err := i.client.Create(ctx, expectedSs)
		if err != nil {
			i.eventBus.Status.MarkDeployFailed("CreateStreamingStatefulSetFailed", "Failed to create a streaming statefulset")
			log.Error(err, "error creating a streaming statefulset")
			return err
		}
		log.Info("streaming statefulset is created", "statefulsetName", expectedSs.Name)
	}
	return nil
}

// buildServerService builds a Service for NATS server
func (i *natsInstaller) buildServerService() (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateServerServiceName(i.eventBus),
			Namespace: i.eventBus.Namespace,
			Labels:    i.componentLabels(componentServer),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{Name: "client", Port: clientPort},
				{Name: "cluster", Port: clusterPort},
				{Name: "monitor", Port: monitorPort},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: i.componentLabels(componentServer),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return svc, nil
}

// buildStreamingService builds a Service for NATS streaming
func (i *natsInstaller) buildStreamingService() (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateStreamingServiceName(i.eventBus),
			Namespace: i.eventBus.Namespace,
			Labels:    i.componentLabels(componentStreaming),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Ports: []corev1.ServicePort{
				{Name: "metrics", Port: metricsPort},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: i.componentLabels(componentStreaming),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, svc, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return svc, nil
}

// buildServerConfigMap builds a ConfigMap for NATS server configuration
func (i *natsInstaller) buildServerConfigMap() (*corev1.ConfigMap, error) {
	routes := ""
	size := i.eventBus.Spec.NATS.Native.Size
	if size <= 0 {
		size = 1
	}
	ssName := generateServerStatefulSetName(i.eventBus)
	svcName := generateServerServiceName(i.eventBus)
	for j := 0; j < size; j++ {
		routes += fmt.Sprintf("\n    nats://%s-%s.%s.%s.svc:%s", ssName, strconv.Itoa(j), svcName, i.eventBus.Namespace, strconv.Itoa(int(clusterPort)))
	}
	conf := fmt.Sprintf(`pid_file: "/var/run/nats/nats.pid"
port: %s
monitor_port: %s
include ./auth.conf
cluster {
  port: %s
  routes: [%s
  ]
  cluster_advertise: $CLUSTER_ADVERTISE
  connect_retries: 30
}`, strconv.Itoa(int(clientPort)), strconv.Itoa(int(monitorPort)), strconv.Itoa(int(clusterPort)), routes)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateServerConfigMapName(i.eventBus),
			Labels:    i.componentLabels(componentServer),
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

// buildStreamingConfigMap builds a ConfigMap for NATS streaming
func (i *natsInstaller) buildStreamingConfigMap() (*corev1.ConfigMap, error) {
	clusterID := generateStreamingClusterID(i.eventBus)
	ssName := generateStreamingStatefulSetName(i.eventBus)
	svcName := generateServerServiceName(i.eventBus)
	peers := fmt.Sprintf("[\"%s-0\", \"%s-1\", \"%s-2\"]", ssName, ssName, ssName)
	conf := fmt.Sprintf(`port: %s
monitor_port: %s
streaming {
  cluster_id: %s
  nats_server_url: "nats://%s:%s"
  store: file
  dir: /data/stan/store
  cluster {
	node_id: $POD_NAME
	peers: %s
  }
}`, strconv.Itoa(int(clientPort)), strconv.Itoa(int(monitorPort)), clusterID, svcName, strconv.Itoa(int(clientPort)), peers)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateStreamingConfigMapName(i.eventBus),
			Labels:    i.componentLabels(componentStreaming),
		},
		Data: map[string]string{
			streamingConfigMapKey: conf,
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
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   i.eventBus.Namespace,
			Name:        generateServerAuthSecretName(i.eventBus),
			Labels:      serverAuthSecretLabels(i.componentLabels(componentServer)),
			Annotations: map[string]string{authStrategyAnnoKey: string(authStrategy)},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			serverAuthSecretKey: []byte(secret),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, s, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return s, nil
}

// buildClientAuthSecret builds a secret for NATS client auth
func (i *natsInstaller) buildClientAuthSecret(authStrategy v1alpha1.AuthStrategy, secret string) (*corev1.Secret, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   i.eventBus.Namespace,
			Name:        generateClientAuthSecretName(i.eventBus),
			Labels:      clientAuthSecretLabels(i.componentLabels(componentServer)),
			Annotations: map[string]string{authStrategyAnnoKey: string(authStrategy)},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			clientAuthSecretKey: []byte(secret),
		},
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, s, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return s, nil
}

// buildServerStatefulSet builds a StatefulSet for NATS server
func (i *natsInstaller) buildServerStatefulSet(serviceName, configmapName, authSecretName string) (*appv1.StatefulSet, error) {
	ss := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateServerStatefulSetName(i.eventBus),
			Labels:    i.componentLabels(componentServer),
		},
		// Use provided serviceName, configMapName and secretName to build the spec
		// to avoid issues when naming convention changes
		Spec: i.buildServerStatefulSetSpec(serviceName, configmapName, authSecretName),
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, ss, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return ss, nil
}

func (i *natsInstaller) buildServerStatefulSetSpec(serviceName, configmapName, authSecretName string) appv1.StatefulSetSpec {
	size := int32(i.eventBus.Spec.NATS.Native.Size)
	if size == 0 {
		size = 1
	}
	l := i.componentLabels(componentServer)
	terminationGracePeriodSeconds := int64(60)
	spec := appv1.StatefulSetSpec{
		Replicas:    &size,
		ServiceName: serviceName,
		Selector: &metav1.LabelSelector{
			MatchLabels: l,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: l,
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
						Image: i.natsImage,
						Ports: []corev1.ContainerPort{
							{Name: "client", ContainerPort: clientPort},
							{Name: "cluster", ContainerPort: clusterPort},
							{Name: "monitor", ContainerPort: monitorPort},
						},
						Command: []string{"/nats-server", "--config", "/etc/nats-config/nats.conf"},
						Env: []corev1.EnvVar{
							{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
							{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
							{Name: "CLUSTER_ADVERTISE", Value: "$(POD_NAME)." + generateServerServiceName(i.eventBus) + ".$(POD_NAMESPACE).svc"},
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
	if i.eventBus.Spec.NATS.Native.Affinity {
		spec.Template.Spec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						TopologyKey: "kubernetes.io/hostname",
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: l,
						},
					},
				},
			},
		}
	}
	return spec
}

// buildStreamingStatefulSet builds a StatefulSet for nats streaming
func (i *natsInstaller) buildStreamingStatefulSet(serviceName, configmapName string) (*appv1.StatefulSet, error) {
	ss := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.eventBus.Namespace,
			Name:      generateStreamingStatefulSetName(i.eventBus),
			Labels:    i.componentLabels(componentStreaming),
		},
		// Use provided serviceName, configMapName to build the spec
		// to avoid issues when naming convention changes
		Spec: i.buildStreamingStatefulSetSpec(serviceName, configmapName),
	}
	if err := controllerscommon.SetObjectMeta(i.eventBus, ss, v1alpha1.SchemaGroupVersionKind); err != nil {
		return nil, err
	}
	return ss, nil
}

func (i *natsInstaller) buildStreamingStatefulSetSpec(serviceName, configmapName string) appv1.StatefulSetSpec {
	// Streaming requires minimal size 3.
	replicas := int32(i.eventBus.Spec.NATS.Native.Size)
	if replicas < 3 {
		replicas = 3
	}
	l := i.componentLabels(componentStreaming)
	spec := appv1.StatefulSetSpec{
		Replicas:    &replicas,
		ServiceName: serviceName,
		Selector: &metav1.LabelSelector{
			MatchLabels: l,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: l,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: configmapName,
								},
								Items: []corev1.KeyToPath{
									{
										Key:  streamingConfigMapKey,
										Path: "stan.conf",
									},
								},
							},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:  "stan",
						Image: i.streamingImage,
						Ports: []corev1.ContainerPort{
							{Name: "monitor", ContainerPort: monitorPort},
							{Name: "metrics", ContainerPort: metricsPort},
						},
						Command: []string{"/nats-streaming-server", "-sc", "/etc/stan-config/stan.conf"},
						Env: []corev1.EnvVar{
							{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
							{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "config-volume", MountPath: "/etc/stan-config"},
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: apiresource.MustParse("0"),
							},
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
					},
				},
			},
		},
	}
	if i.eventBus.Spec.NATS.Native.Persistence != nil {
		volMode := corev1.PersistentVolumeFilesystem
		pvcName := generatePVCName(i.eventBus)
		spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: pvcName,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					VolumeMode:       &volMode,
					StorageClassName: i.eventBus.Spec.NATS.Native.Persistence.StorageClassName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: apiresource.MustParse("1Gi"),
						},
					},
				},
			},
		}
		volumes := spec.Template.Spec.Containers[0].VolumeMounts
		volumes = append(volumes, corev1.VolumeMount{Name: pvcName, MountPath: "/data/stan"})
		spec.Template.Spec.Containers[0].VolumeMounts = volumes
	}
	if i.eventBus.Spec.NATS.Native.Affinity {
		spec.Template.Spec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						TopologyKey: "kubernetes.io/hostname",
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: l,
						},
					},
				},
			},
		}
	}
	return spec
}

func (i *natsInstaller) getServerService(ctx context.Context) (*corev1.Service, error) {
	return i.getService(ctx, i.componentLabels(componentServer))
}

func (i *natsInstaller) getStreamingService(ctx context.Context) (*corev1.Service, error) {
	return i.getService(ctx, i.componentLabels(componentStreaming))
}

func (i *natsInstaller) getService(ctx context.Context, labels map[string]string) (*corev1.Service, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	sl := &corev1.ServiceList{}
	err := i.client.List(ctx, sl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(labels),
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

func (i *natsInstaller) getServerConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	return i.getConfigMap(ctx, i.componentLabels(componentServer))
}

func (i *natsInstaller) getStreamingConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	return i.getConfigMap(ctx, i.componentLabels(componentStreaming))
}

func (i *natsInstaller) getConfigMap(ctx context.Context, labels map[string]string) (*corev1.ConfigMap, error) {
	cml := &corev1.ConfigMapList{}
	err := i.client.List(ctx, cml, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(labels),
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
	return i.getSecret(ctx, serverAuthSecretLabels(i.componentLabels(componentServer)))
}

// get client auth secret
func (i *natsInstaller) getClientAuthSecret(ctx context.Context) (*corev1.Secret, error) {
	return i.getSecret(ctx, clientAuthSecretLabels(i.componentLabels(componentServer)))
}

func (i *natsInstaller) getSecret(ctx context.Context, labels map[string]string) (*corev1.Secret, error) {
	sl := &corev1.SecretList{}
	err := i.client.List(ctx, sl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(labels),
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

func (i *natsInstaller) getServerStatefulSet(ctx context.Context) (*appv1.StatefulSet, error) {
	return i.getStatefulSet(ctx, i.componentLabels(componentServer))
}

func (i *natsInstaller) getStreamingStatefulSet(ctx context.Context) (*appv1.StatefulSet, error) {
	return i.getStatefulSet(ctx, i.componentLabels(componentStreaming))
}

func (i *natsInstaller) getStatefulSet(ctx context.Context, labels map[string]string) (*appv1.StatefulSet, error) {
	// Why not using getByName()?
	// Naming convention might be changed.
	ssl := &appv1.StatefulSetList{}
	err := i.client.List(ctx, ssl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(labels),
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

// get PVCs created by streaming statefulset
// they have same labels as the statefulset
func (i *natsInstaller) getPVCs(ctx context.Context, labels map[string]string) ([]corev1.PersistentVolumeClaim, error) {
	pvcl := &corev1.PersistentVolumeClaimList{}
	err := i.client.List(ctx, pvcl, &client.ListOptions{
		Namespace:     i.eventBus.Namespace,
		LabelSelector: labelSelector(labels),
	})
	if err != nil {
		return nil, err
	}
	return pvcl.Items, nil
}

func (i *natsInstaller) componentLabels(c component) map[string]string {
	result := make(map[string]string)
	for k, v := range i.labels {
		result[k] = v
	}
	result[componentLabelKey] = string(c)
	return result
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

func serverAuthSecretLabels(given map[string]string) map[string]string {
	result := map[string]string{"server-auth-secret": "yes"}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func clientAuthSecretLabels(given map[string]string) map[string]string {
	result := map[string]string{"client-auth-secret": "yes"}
	for k, v := range given {
		result[k] = v
	}
	return result
}

func labelSelector(labelMap map[string]string) labels.Selector {
	return labels.SelectorFromSet(labelMap)
}

func generateServerServiceName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-svc", eventBus.Name)
}

func generateStreamingServiceName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-stan-svc", eventBus.Name)
}

func generateServerConfigMapName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-configmap", eventBus.Name)
}

func generateStreamingConfigMapName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-stan-configmap", eventBus.Name)
}

func generateServerAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-server", eventBus.Name)
}

func generateClientAuthSecretName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-client", eventBus.Name)
}

func generateServerStatefulSetName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s", eventBus.Name)
}

func generateStreamingStatefulSetName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus-%s-stan", eventBus.Name)
}

func generateStreamingClusterID(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("eventbus_%s", eventBus.Name)
}

// PVC name used in streaming statefulset
func generatePVCName(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("stan-%s-vol", eventBus.Name)
}

// Final PVC name prefix
func getFinalizedPVCNamePrefix(eventBus *v1alpha1.EventBus) string {
	return fmt.Sprintf("%s-%s-", generatePVCName(eventBus), generateStreamingStatefulSetName(eventBus))
}
